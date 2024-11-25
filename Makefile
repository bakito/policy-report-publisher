pf-hubble-443:
	kubectl port-forward -n kube-system svc/hubble-relay 32766:443
pf-hubble-80:
	kubectl port-forward -n kube-system svc/hubble-relay 32766:80
pf-kubearmor:
	kubectl port-forward -n kubearmor svc/kubearmor 32767:32767