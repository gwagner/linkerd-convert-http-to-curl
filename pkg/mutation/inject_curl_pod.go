package mutation

import (
	"fmt"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

// injectEnv is a container for the mutation injecting environment vars
type injectCurl struct {
	Logger logrus.FieldLogger
}

// injectEnv implements the podMutator interface
var _ podMutator = (*injectCurl)(nil)

// Name returns the struct name
func (se injectCurl) Name() string {
	return "inject_curl"
}

// Mutate returns a new mutated pod according to set env rules
func (se injectCurl) Mutate(pod *corev1.Pod) (*corev1.Pod, error) {
	se.Logger = se.Logger.WithField("mutation", se.Name())
	mpod := pod.DeepCopy()

	se.injectCurlPod(mpod)

	return mpod, nil
}

// injectCurlPod injects a var in both containers and init containers of a pod
func (se injectCurl) injectCurlPod(pod *corev1.Pod) {
	curlContainer := corev1.Container{
		Command:                  []string{"sleep", "365d"},
		Image:                    "rancher/curl:latest",
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "FallbackToLogsOnError",
	}

	// if this is not a linkerd pod, do nothing
	if _, ok := pod.Annotations["linkerd.io/proxy-version"]; !ok {
		se.Logger.Debugf("%s Does not use linkerd, skipping", pod.Name)
		return
	}

	// if this is not a linkerd pod, do nothing
	if _, ok := pod.Annotations["valewood.org/local-curl-probe"]; ok {
		se.Logger.Debug("Local curl probes already injected, skipping")
		return
	}

	for i, container := range pod.Spec.Containers {

		// if we dont actually have any http probes, no need to do a conversion and we can bail
		if container.LivenessProbe == nil && container.ReadinessProbe == nil {
			se.Logger.Debugf("No readiness or liveness probes found, skipping %s", container.Name)
			continue
		}

		// if we dont actually have any http probes, no need to do a conversion and we can bail
		if container.LivenessProbe.Handler.HTTPGet == nil && container.ReadinessProbe.Handler.HTTPGet == nil {
			se.Logger.Debugf("No http readiness or liveness probes found, skipping %s", container.Name)
			continue
		}

		// If we have an HTTP Get liveness probe, we need to inject a curl pod
		if container.LivenessProbe.Handler.HTTPGet != nil {
			se.Logger.Debugf("Replacing liveness probes for %s", container.Name)

			// Build out the new exec probe for the curl container
			curlContainer.LivenessProbe = buildExecProbe(container.LivenessProbe)

			// Unset the liveness probe because it was replaced by a curl container
			pod.Spec.Containers[i].LivenessProbe = nil
		}

		// If we have an HTTP Get liveness probe, we need to inject a curl pod
		if container.ReadinessProbe.Handler.HTTPGet != nil {
			se.Logger.Debugf("Replacing readiness probes for %s", container.Name)

			// Build out the new exec probe for the curl container
			curlContainer.ReadinessProbe = buildExecProbe(container.ReadinessProbe)

			// Unset the liveness probe because it was replaced by a curl container
			pod.Spec.Containers[i].ReadinessProbe = nil
		}

		curlContainer.Name = "local-curl-probe-" + container.Name

		// Add the container to the pod to replace health checks
		se.Logger.Debugf("Adding new container %s", curlContainer.Name)
		pod.Spec.Containers = append(pod.Spec.Containers, curlContainer)
		pod.Annotations["valewood.org/local-curl-probe"] = "yes"
	}
}

func buildExecCommand(probe *corev1.HTTPGetAction) *corev1.ExecAction {
	host := probe.Host
	if host == "" {
		host = "127.0.0.1"
	}

	port := probe.Port.IntVal
	if port == 0 {
		port = 80
	}

	execAction := &corev1.ExecAction{
		Command: []string{
			"curl",
			fmt.Sprintf("%s://%s:%d%s",
				strings.ToLower(string(probe.Scheme)),
				host,
				port,
				probe.Path),
			"--fail",
			"-o",
			"/dev/null",
		},
	}

	execAction.Command = append(execAction.Command, httpGetProbeHeadersToCurl(probe.HTTPHeaders)...)

	return execAction
}

func buildExecProbe(probe *corev1.Probe) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: buildExecCommand(probe.Handler.HTTPGet),
		},
		InitialDelaySeconds:           probe.InitialDelaySeconds,
		TimeoutSeconds:                probe.TimeoutSeconds,
		PeriodSeconds:                 probe.PeriodSeconds,
		SuccessThreshold:              probe.SuccessThreshold,
		FailureThreshold:              probe.FailureThreshold,
		TerminationGracePeriodSeconds: probe.TerminationGracePeriodSeconds,
	}
}

func httpGetProbeHeadersToCurl(headers []corev1.HTTPHeader) []string {
	ret := []string{}

	for _, v := range headers {
		ret = append(ret, "-H", fmt.Sprintf("%s: %s", v.Name, v.Value))
	}

	return ret
}
