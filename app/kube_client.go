package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"net/http"
	"net/url"
	"sync"
)

type EventType string

const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
)

type ObjectEvent struct {
	Type   EventType   `json:"type"`
	Object *KubeObject `json:"object"`
}

type ObjectList struct {
	Objects []KubeObject `json:"items"`
}

type KubeClient interface {
	Server() KubeServer
	Ping() error
	WatchObjects(kind string, out chan *ObjectEvent) error
	GetObjects(kind string) ([]KubeObject, error)
}

type DefaultKubeClient struct {
	client  *http.Client
	baseURL *url.URL
	kcClient kubernetes.Interface
}

//NewKubeClient returns a client that talks to Kubenetes API server.
//It talks to only one server, and uses configuration of the current context in the
//given config
//TODO return the error rather than log.Fatal
func NewKubeClient(config *Config) KubeClient {
	tlsConfig, _ := config.GenerateTLSConfig()
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	httpClient := &http.Client{Transport: tr}

	url, _ := url.Parse(config.getCurrentCluster().Server)

	dkc := &DefaultKubeClient{
		client:  httpClient,
		baseURL: url,
		//kcClient: clientset,
	}
	if config.CurrentClient != nil {
		clientset, err := kubernetes.NewForConfig(config.CurrentClient)
		if err != nil {
			log.Fatal(err)
		}
		dkc.kcClient = clientset
	}
	return dkc
}

func (kc *DefaultKubeClient) Server() KubeServer {
	return KubeServer{kc.baseURL.String()}
}

func (kc *DefaultKubeClient) Ping() error {
	nodes, err := kc.kcClient.CoreV1().Nodes().List(metav1.ListOptions{})

	if err != nil {
		return err
	}
	log.Debugf("Nodes: %s", nodes)
	if len(nodes.Items) < 1 {
		return fmt.Errorf("No nodes available")
	}

	return nil
}

func (kc *DefaultKubeClient) WatchObjects(kind string, out chan *ObjectEvent) error {
	switch kind {
	case "pod":
		//return kc.watch("api/v1/pods?watch=true", out)
		log.Debugf("case pod")
		ch, err := kc.watchPods(v1.NamespaceAll)
		if err != nil {
			return fmt.Errorf("Watching pods failed: %s", err)
		}
		for event := range ch {
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				return fmt.Errorf("Unexpected type: %s", pod)
			}
			var evt ObjectEvent
			log.Debug(event.Type)
			switch event.Type {
			case watch.Added:
				evt.Type = Added
			case watch.Deleted:
				evt.Type = Deleted
			case watch.Modified:
				evt.Type = Modified
			}
			evt.Object = &KubeObject{
				TypeMeta:   TypeMeta{Kind: "pod"},
				ObjectMeta: ObjectMeta{
					Name: pod.Name,
					Namespace: pod.Namespace,
					ResourceVersion: pod.ResourceVersion,
				},
			}
			out <- &evt
		}
	case "service":
		log.Debug("case service")
		ch, err := kc.watchServices(v1.NamespaceAll)
		if err != nil {
			return fmt.Errorf("Watching services failed: %s", err)
		}
		for event := range ch {
			svc, ok := event.Object.(*v1.Service)
			if !ok {
				return fmt.Errorf("Unexpected type: %s", svc)
			}
			var evt ObjectEvent
			log.Debug(event.Type)
			switch event.Type {
			case watch.Added:
				evt.Type = Added
			case watch.Deleted:
				evt.Type = Deleted
			case watch.Modified:
				evt.Type = Modified
			}
			evt.Object = &KubeObject{
				TypeMeta:   TypeMeta{Kind: "service"},
				ObjectMeta: ObjectMeta{
					Name: svc.Name,
					Namespace: svc.Namespace,
					ResourceVersion: svc.ResourceVersion,
				},
			}
			out <- &evt

		}
	case "deployment":
		return kc.watch("/apis/extensions/v1beta1/deployments?watch=true", out)
	default:
		return fmt.Errorf("unsupported kind: %s", kind)
	}

	return nil
}

func (kc *DefaultKubeClient) GetObjects(kind string) ([]KubeObject, error) {
	var objects []KubeObject
	switch kind {
	case "node":
		nodes, err := kc.getNodes()
		if err != nil {
			return []KubeObject{}, err
		}
		log.Debug(nodes)
		for _, node := range nodes.Items {
			addToKubeObjects(&objects, "node", node.Name, node.Namespace, node.ResourceVersion)
		}
		return objects, nil
	case "configmap":
		cms, err := kc.getConfigMaps(v1.NamespaceAll)
		if err != nil {
			return []KubeObject{}, err
		}
		log.Debug(cms)

		for _, cm := range cms.Items {
			addToKubeObjects(&objects, "configmap", cm.Name, cm.Namespace, cm.ResourceVersion)
		}
		return objects, nil
	case "service":
		svc, err := kc.getServices(v1.NamespaceAll)
		if err != nil {
			return []KubeObject{}, err
		}
		log.Debug(svc)

		for _, sv := range svc.Items {
			addToKubeObjects(&objects, "service", sv.Name, sv.Namespace, sv.ResourceVersion)
		}
		return objects, nil

	case "deployment":
		deployment, err := kc.getDeployments(v1.NamespaceAll)
		if err != nil {
			return []KubeObject{}, err
		}
		log.Debug(deployment)

		for _, d := range deployment.Items {
			addToKubeObjects(&objects, "deployment", d.Name, d.Namespace, d.ResourceVersion)
		}
		return objects, nil
	case "namespace":
		ns, err := kc.getNamespaces()
		if err != nil {
			return []KubeObject{}, err
		}
		log.Debug(ns)

		for _, n := range ns.Items {
			addToKubeObjects(&objects, "namespace", n.Name, n.Namespace, n.ResourceVersion)
		}
		return objects, nil
	default:
		return []KubeObject{}, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func addToKubeObjects(objectsPtr *[]KubeObject, tm , name, ns, rv string ) {
	//NOTE: need to pass the KubeObject as a pointer because I'm altering the actual slice by appending to it
	objects := *objectsPtr
	t := TypeMeta{Kind: tm}
	*objectsPtr = append(objects, KubeObject{
		TypeMeta:   t,
		ObjectMeta: ObjectMeta{
			Name:            name,
			Namespace:       ns,
			ResourceVersion: rv,
		},
	})

}

func (kc *DefaultKubeClient) getNodes() (*v1.NodeList, error) {
	return kc.kcClient.CoreV1().Nodes().List(metav1.ListOptions{})
}

func (kc *DefaultKubeClient) getConfigMaps(ns string) (*v1.ConfigMapList, error) {
	return kc.kcClient.CoreV1().ConfigMaps(ns).List(metav1.ListOptions{})
}

func (kc *DefaultKubeClient) getServices(ns string) (*v1.ServiceList, error) {
	return kc.kcClient.CoreV1().Services(ns).List(metav1.ListOptions{})
}

func (kc *DefaultKubeClient) watchServices(ns string) (out <-chan watch.Event, err error) {
	req, err := kc.kcClient.CoreV1().Services(ns).Watch(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return req.ResultChan(), nil
}

func (kc *DefaultKubeClient) watchPods(ns string) (out <-chan watch.Event, err error) {
	req, err := kc.kcClient.CoreV1().Pods(ns).Watch(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return req.ResultChan(), nil
}

func (kc *DefaultKubeClient) getDeployments(ns string) (*v1apps.DeploymentList, error) {
	return kc.kcClient.AppsV1().Deployments(ns).List(metav1.ListOptions{})
}

func (kc *DefaultKubeClient) getNamespaces() (*v1.NamespaceList, error) {
	return kc.kcClient.CoreV1().Namespaces().List(metav1.ListOptions{})
}

//func (kc *DefaultKubeClient) get(url string, kind string) ([]KubeObject, error) {
//	req, err := kc.newRequest("GET", url, nil)
//	if err != nil {
//		return []KubeObject{}, err
//	}
//
//	var list ObjectList
//	err = kc.do(req, &list)
//	if err != nil {
//		return []KubeObject{}, err
//	}
//
//	for i := range list.Objects {
//		list.Objects[i].Kind = kind
//	}
//
//	return list.Objects, nil
//}

func (kc *DefaultKubeClient) watch(url string, out chan *ObjectEvent) error {
	req, err := kc.newRequest("GET", url, nil)
	if err != nil {
		return err
	}

	res, err := kc.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to watch pods: %d", res.StatusCode)
	}

	d := json.NewDecoder(res.Body)

	for {
		var event ObjectEvent
		err := d.Decode(&event)

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Could not decode data into pod event: %s", err)
		}

		out <- &event
	}

	return nil
}

func (kc *DefaultKubeClient) newRequest(method string, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	u := kc.baseURL.ResolveReference(rel)
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *DefaultKubeClient) do(req *http.Request, v interface{}) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		// Drain up to 512 bytes and close the body to let the Transport reuse the connection
		io.CopyN(ioutil.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			body = []byte(err.Error())
		}
		return fmt.Errorf("unexpected status for %s %s: %s %s", req.Method, req.URL, resp.Status, string(body))
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
		if err == io.EOF {
			err = nil // ignore EOF errors caused by empty response body
		}
	}

	return err
}

type TestKubeClient struct {
	baseURL *url.URL
	pings   int

	objectEvents  []*ObjectEvent
	objectEventsF func() []*ObjectEvent

	watchObjectHits  map[string]int
	watchObjectLock  *sync.RWMutex
	watchObjectError error

	objects       []KubeObject
	objectsF      func() []KubeObject
	getObjectHits map[string]int
	kcClient kubernetes.Interface
}

func NewTestKubeClient() *TestKubeClient {
	kc := &TestKubeClient{}
	//kc.baseURL, _ = url.Parse(fmt.Sprintf("http://random-url-%d.com", rand.Intn(999)))
	kc.baseURL, _ = url.Parse("https://bar.com")
	kc.watchObjectLock = &sync.RWMutex{}
	kc.watchObjectHits = map[string]int{}
	kc.objectEventsF = func() []*ObjectEvent { return []*ObjectEvent{} }
	kc.objects = []KubeObject{}
	kc.objectsF = func() []KubeObject { return []KubeObject{} }
	kc.getObjectHits = map[string]int{}
	kc.kcClient = testclient.NewSimpleClientset()
	return kc
}

func (kc *TestKubeClient) Server() KubeServer {
	return KubeServer{kc.baseURL.String()}
}

func (kc *TestKubeClient) Ping() error {
	kc.pings += 1
	return nil
}

func (kc *TestKubeClient) WatchObjects(kind string, out chan *ObjectEvent) error {
	kc.watchObjectLock.Lock()
	kc.watchObjectHits[kind] += 1
	kc.watchObjectLock.Unlock()

	for i := range kc.objectEvents {
		out <- kc.objectEvents[i]
	}

	for _, o := range kc.objectEventsF() {
		out <- o
	}

	if kc.watchObjectHits[kind] < 5 && kc.watchObjectError != nil {
		return kc.watchObjectError
	}

	select {}
}

func (kc *TestKubeClient) GetObjects(kind string) ([]KubeObject, error) {
	kc.watchObjectLock.Lock()
	kc.getObjectHits[kind] += 1
	kc.watchObjectLock.Unlock()

	if len(kc.objects) == 0 {
		return kc.objectsF(), nil
	} else {
		return kc.objects, nil
	}
}
