# linkerd-convert-http-to-curl

> Code was originally pulled from here: https://github.com/slackhq/simple-kubernetes-webhook
> This follows the original license from the original repo regardless of how this repo may be licensed (in case this repo falls behind the original repo).

This is a simple webhook implementation which takes any http liveness or readiness probes and converts them to an exec 
curl commands in a supplemental container of a pod.  This lets you get around mtls limitations with linkerd and istio by
not needing to compromise security by being overly permissive to allow probes to pass.  This also allows your
developers to work transparently by writing their configurations using http liveness and readiness probes and you (as the admin) 
get to bypass any sort of retraining effort while locking down your cluster.

Cool right?!?