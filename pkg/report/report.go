package report

import (
	"context"

	prv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var scheme = runtime.NewScheme()

func NewKubeClient() (error, client.Client) {
	utilruntime.Must(prv1alpha2.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	restClientGetter := genericclioptions.ConfigFlags{}
	rawKubeConfigLoader := restClientGetter.ToRawKubeConfigLoader()

	config, err := rawKubeConfigLoader.ClientConfig()
	if err != nil {
		panic(err)
	}

	kc, err := client.New(config, client.Options{Scheme: scheme})
	return err, kc
}

func Update(ctx context.Context, kc client.Client, podNamespace string, podName string,
	mutate func(pol *prv1alpha2.PolicyReport) error,
) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		pol, err := getPolicyReport(ctx, kc, podNamespace, podName)
		if err != nil {
			return err
		}
		_, err = controllerutil.CreateOrUpdate(ctx, kc, pol, func() error {
			return mutate(pol)
		})
		return err
	})
}

func getPolicyReport(ctx context.Context, kc client.Client, podNamespace string, podName string) (*prv1alpha2.PolicyReport, error) {

	pod := &corev1.Pod{}
	err := kc.Get(ctx, types.NamespacedName{Namespace: podNamespace, Name: podName}, pod)
	if err != nil {
		return nil, nil
	}

	pol := &prv1alpha2.PolicyReport{}
	err = kc.Get(ctx, types.NamespacedName{Namespace: pod.GetNamespace(), Name: string(pod.GetUID())}, pol)

	if err != nil {
		if errors.IsNotFound(err) {
			pol = &prv1alpha2.PolicyReport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: pod.GetNamespace(),
					Name:      string(pod.GetUID()),
				},
			}

			_ = controllerutil.SetOwnerReference(pod, pol, kc.Scheme())
			pol.Scope = &corev1.ObjectReference{
				Namespace:  pod.Namespace,
				Name:       pod.Name,
				Kind:       pod.Kind,
				UID:        pod.GetUID(),
				APIVersion: pod.APIVersion,
			}
		} else {
			return nil, err
		}
	}

	return pol, nil
}
