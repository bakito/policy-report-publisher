package report

// https://github.com/kubernetes/client-go/blob/master/examples/leader-election/main.go

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/bakito/policy-report-publisher/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

var (
	defaultLeaseDuration = 15 * time.Second
	defaultRenewDeadline = 10 * time.Second
	defaultRetryPeriod   = 2 * time.Second
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete,namespace="{{.Release.Namespace}}"

func (h *handler) RunAsLeader(ctx context.Context, cancel context.CancelFunc, leaseLockNamespace string,
	run func(ctx context.Context, handler Handler, cancel context.CancelFunc),
) error {
	// Leader id, needs to be unique
	id, err := os.Hostname()
	if err != nil {
		return err
	}
	id = id + "_" + string(uuid.NewUUID())

	// leader election uses the Kubernetes API by writing to a
	// lock object, which can be a LeaseLock object (preferred),
	// a ConfigMap, or an Endpoints (deprecated) object.
	// Conflicting writes are detected and each client handles those actions
	// independently.

	// we use the Lease lock type since edits to Leases are less common
	// and fewer objects in the cluster watch "all Leases".
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      version.Name,
			Namespace: leaseLockNamespace,
		},
		Client: h.clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   defaultLeaseDuration,
		RenewDeadline:   defaultRenewDeadline,
		RetryPeriod:     defaultRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				run(ctx, h, cancel)
			},
			OnStoppedLeading: func() {
				// we can do cleanup here, but note that this callback is always called
				// when the LeaderElector exits, even if it did not start leading.
				// Therefore, we should check if we actually started leading before
				// performing any cleanup operations to avoid unexpected behavior.
				slog.Info("leader lost", "identity", id)

				os.Exit(0)
			},

			OnNewLeader: func(identity string) {
				// we're notified when new leader elected
				if identity == id {
					// I just got the lock
					return
				}
				slog.Info("new leader elected", "identity", identity)
			},
		},
	})
	return nil
}
