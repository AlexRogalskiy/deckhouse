package deckhouse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/client"
	"github.com/deckhouse/deckhouse/candictl/pkg/log"
)

var (
	ErrListPods      = errors.New("No Deckhouse pod found.")
	ErrTimedOut      = errors.New("Time is out waiting for Deckhouse readiness.")
	ErrRequestFailed = errors.New("Request failed. Probably pod doesn't exist anymore.")
)

type logLine struct {
	Module    string `json:"module,omitempty"`
	Level     string `json:"level,omitempty"`
	Output    string `json:"output,omitempty"`
	Message   string `json:"msg,omitempty"`
	Component string `json:"operator.component,omitempty"`
}

func printLogsByLine(content []byte) {
	reader := bufio.NewReader(bytes.NewReader(content))
	for {
		l, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		var line logLine
		if err := json.Unmarshal(l, &line); err != nil {
			continue
		}

		if line.Level == "error" || (line.Output == "stderr" && line.Component != "tiller") {
			log.ErrorF("\t%s\n", line.Message)
			continue
		}

		// TODO use module.state label
		if line.Message == "Module run success" || line.Message == "ModuleRun success, module is ready" {
			log.InfoF("\tModule %q run successfully\n", line.Module)
			continue
		}

		if line.Message == "Queue 'main' contains 0 converge tasks after handle 'ModuleHookRun'" {
			log.InfoLn("No more converge tasks found in Deckhouse queue.")
		}
	}
}

type LogPrinter struct {
	kubeCl *client.KubernetesClient

	deckhousePod       *corev1.Pod
	waitPodBecomeReady bool
}

func NewLogPrinter(kubeCl *client.KubernetesClient) *LogPrinter {
	return &LogPrinter{kubeCl: kubeCl}
}

func (d *LogPrinter) WaitPodBecomeReady() *LogPrinter {
	d.waitPodBecomeReady = true
	return d
}

func (d *LogPrinter) GetPod() error {
	pods, err := d.kubeCl.CoreV1().Pods("d8-system").List(metav1.ListOptions{LabelSelector: "app=deckhouse"})
	if err != nil {
		return ErrListPods
	}

	if len(pods.Items) != 1 {
		return ErrListPods
	}

	pod := pods.Items[0]

	message := fmt.Sprintf("Deckhouse pod found: %s (%s)", pod.Name, pod.Status.Phase)
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf(message)
	}

	log.InfoLn(message)
	log.InfoLn("Running pod found! Checking logs...")

	d.deckhousePod = &pod
	return nil
}

func (d *LogPrinter) checkDeckhousePodReady() (bool, error) {
	if !d.waitPodBecomeReady || d.deckhousePod == nil {
		return false, nil
	}

	runningPod, err := d.kubeCl.CoreV1().Pods("d8-system").Get(d.deckhousePod.Name, metav1.GetOptions{})
	if err != nil {
		return false, ErrRequestFailed
	}

	ready := true
	for _, condition := range runningPod.Status.Conditions {
		if condition.Status == corev1.ConditionTrue {
			continue
		}
		ready = false
		log.DebugF("Pod %s is not ready: %s = %s\n", d.deckhousePod.Name, condition.Type, condition.Status)
	}

	return ready, nil
}

func (d *LogPrinter) Print(ctx context.Context) (bool, error) {
	if err := d.GetPod(); err != nil {
		return false, err
	}

	logOptions := corev1.PodLogOptions{Container: "deckhouse", TailLines: int64Pointer(5)}
	defer func() { d.deckhousePod = nil }()

	for {
		select {
		case <-ctx.Done():
			return false, ErrTimedOut
		default:
			request := d.kubeCl.CoreV1().Pods("d8-system").GetLogs(d.deckhousePod.Name, &logOptions)
			result, err := request.DoRaw()
			if err != nil {
				log.DebugLn(err)
				return false, ErrRequestFailed
			}

			printLogsByLine(result)
			ready, err := d.checkDeckhousePodReady()
			if err != nil {
				return false, err
			}
			if ready {
				return true, nil
			}

			time.Sleep(time.Second)
			currentTime := metav1.NewTime(time.Now())
			logOptions = corev1.PodLogOptions{Container: "deckhouse", SinceTime: &currentTime}
		}
	}
}

func int64Pointer(i int) *int64 {
	r := int64(i)
	return &r
}
