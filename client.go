package trackedclient

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type TrackedClient interface {
	client.WithWatch
	DeleteAllTracked(context.Context, ...client.DeleteOption) error
}

// client is a client.Client that reads and writes directly from/to an API server.  It lazily initializes
// new clients at the time they are used, and caches the client.
type trackedClient struct {
	client.WithWatch
	tracker []unstructured.Unstructured
	lock    *sync.Mutex
}

// New returns a new Client using the provided config and Options.
func New(config *rest.Config, options client.Options) (TrackedClient, error) {
	return newTrackedClient(config, options)
}

func newTrackedClient(config *rest.Config, options client.Options) (*trackedClient, error) {
	c, err := client.NewWithWatch(config, options)
	if err != nil {
		return nil, err
	}
	tc := &trackedClient{
		WithWatch: c,
		lock:      &sync.Mutex{},
	}
	return tc, nil
}

var _ client.Client = &trackedClient{}

// Create implements client.Client
func (c *trackedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	err := c.WithWatch.Create(ctx, obj, opts...)
	if err != nil {
		return err
	}

	// Create an unstructured copy of the object
	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	u := unstructured.Unstructured{Object: objMap}

	// GVK is not always provided
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	u.SetKind(gvk.GroupKind().Kind)
	u.SetAPIVersion(gvk.GroupVersion().String())

	// Store the copy in an indexed tracker to delete later
	c.lock.Lock()
	c.tracker = append(c.tracker, u)
	c.lock.Unlock()
	return nil
}

// remove delete option, force use of precondition to ensure UID matches
func (c *trackedClient) DeleteAllTracked(ctx context.Context, opts ...client.DeleteOption) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var errs []error
	for i := range c.tracker {
		obj := &c.tracker[i]
		uid := obj.GetUID()
		preconditions := client.Preconditions{
			UID: &uid,
		}
		opts = append(opts, preconditions)
		err := c.WithWatch.Delete(ctx, obj, opts...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	c.tracker = nil

	if len(errs) < 1 {
		return nil
	}

	totalErr := errs[0]
	for e := 1; e < len(errs); e++ {
		totalErr = fmt.Errorf("%w; %s", totalErr, errs[e].Error())
	}
	return totalErr
}
