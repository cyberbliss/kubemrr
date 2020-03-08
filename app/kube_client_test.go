package app

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/extensions/v1beta1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	//testclient "k8s.io/client-go/kubernetes/fake"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
	client KubeClient
	testClient KubeClient
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	cfg, _ := NewConfigFromURL(server.URL)
	f := &DefaultFactory{}
	client = f.KubeClient(cfg)
	//tf := &TestFactory{}
	//testClient = tf.KubeClient(cfg)
}

// teardown closes the test HTTP server.
func teardown() {
	server.Close()
}

func stream(w http.ResponseWriter, items []string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("need flusher!")
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for _, item := range items {
		_, err := w.Write([]byte(item))
		if err != nil {
			panic(err)
		}
		flusher.Flush()
	}
}

func TestWatchPods(t *testing.T) {
	events := []interface{}{
		&ObjectEvent{Added, &KubeObject{ObjectMeta: ObjectMeta{Name: "first"}}},
		&ObjectEvent{Modified, &KubeObject{ObjectMeta: ObjectMeta{Name: "second"}}},
		&ObjectEvent{Deleted, &KubeObject{ObjectMeta: ObjectMeta{Name: "last"}}},
	}

	setup()
	defer teardown()
	mux.HandleFunc("/api/v1/pods", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			t.Errorf("URL must have parameter `?watch=true`")
		}
		stream(w, []string{
			`{"type": "ADDED", "object": {"metadata": {"name": "first"}}}`,
			`{"type": "MODIFIED", "object": {"metadata": {"name": "second"}}}`,
			`{"type": "DELETED", "object": {"metadata": {"name": "last"}}}`,
		})
	},
	)

	inEvents := make(chan *ObjectEvent, 10)
	err := client.WatchObjects("pod", inEvents)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	for _, expectedEvent := range events {
		actualEvent := <-inEvents

		if !reflect.DeepEqual(expectedEvent, actualEvent) {
			t.Errorf("Expected %v, received %v", expectedEvent, actualEvent)
		}
	}
}

func TestWatchServices(t *testing.T) {
	log.Info("start")
	events := []interface{}{
		&ObjectEvent{Added, &KubeObject{ObjectMeta: ObjectMeta{Name: "first"}}},
		&ObjectEvent{Modified, &KubeObject{ObjectMeta: ObjectMeta{Name: "second"}}},
		&ObjectEvent{Deleted, &KubeObject{ObjectMeta: ObjectMeta{Name: "last"}}},
	}

	//setup()
	//defer teardown()
	//mux.HandleFunc("/api/v1/services", func(w http.ResponseWriter, r *http.Request) {
	//	if r.URL.Query().Get("watch") != "true" {
	//		t.Errorf("URL must have parameter `?watch=true`")
	//	}
	//	stream(w, []string{
	//		`{"type": "ADDED", "object": {"metadata": {"name": "first"}}}`,
	//		`{"type": "MODIFIED", "object": {"metadata": {"name": "second"}}}`,
	//		`{"type": "DELETED", "object": {"metadata": {"name": "last"}}}`,
	//	})
	//},
	//)

	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	svcs := [][]string{
		{"ns1","svc1"},
		{"ns1","svc2"},
		{"ns2","svc3"},
	}

	log.Debug("here1")
	for _, s := range svcs {
		sv := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: s[1],
			},
		}
		client.kcClient.CoreV1().Services(s[0]).Create(sv)
	}
	inEvents := make(chan *ObjectEvent, 10)
	err := client.WatchObjects("service", inEvents)
	log.Debug("here2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	for _, expectedEvent := range events {
		actualEvent := <-inEvents

		if !reflect.DeepEqual(expectedEvent, actualEvent) {
			t.Errorf("Expected %v, received %v", expectedEvent, actualEvent)
		}
	}
}

func TestWatchDeployments(t *testing.T) {
	events := []interface{}{
		&ObjectEvent{Added, &KubeObject{ObjectMeta: ObjectMeta{Name: "first"}}},
		&ObjectEvent{Modified, &KubeObject{ObjectMeta: ObjectMeta{Name: "second"}}},
		&ObjectEvent{Deleted, &KubeObject{ObjectMeta: ObjectMeta{Name: "last"}}},
	}

	setup()
	defer teardown()
	mux.HandleFunc("/apis/extensions/v1beta1/deployments", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			t.Errorf("URL must have parameter `?watch=true`")
		}
		stream(w, []string{
			`{"type": "ADDED", "object": {"metadata": {"name": "first"}}}`,
			`{"type": "MODIFIED", "object": {"metadata": {"name": "second"}}}`,
			`{"type": "DELETED", "object": {"metadata": {"name": "last"}}}`,
		})
	},
	)

	inEvents := make(chan *ObjectEvent, 10)
	err := client.WatchObjects("deployment", inEvents)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	for _, expectedEvent := range events {
		actualEvent := <-inEvents

		if !reflect.DeepEqual(expectedEvent, actualEvent) {
			t.Errorf("Expected %v, received %v", expectedEvent, actualEvent)
		}
	}
}

func TestGetConfigmaps(t *testing.T) {
	//setup()
	//defer teardown()
	//
	//mux.HandleFunc("/api/v1/configmaps", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprint(w, `
	//		{
	//			"items": [
	//				{ "metadata": { "name": "x1" } },
	//				{ "metadata": { "name": "x2" } }
	//			]
	//		}`)
	//},
	//)

	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	cms := [][]string{
		{"ns1","cm1"},
		{"ns1","cm2"},
		{"ns2","cm3"},
	}
	createTestCMs(client, cms)

	res, err := client.GetObjects("configmap")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := []KubeObject{
		{TypeMeta: TypeMeta{}, ObjectMeta: ObjectMeta{
			Name:      "cm1",
			Namespace: "ns1",
		}},
		{TypeMeta: TypeMeta{Kind: "configmap"}, ObjectMeta: ObjectMeta{Name: "cm2", Namespace: "ns1"}},
		{TypeMeta: TypeMeta{Kind: "configmap"}, ObjectMeta: ObjectMeta{Name: "cm3", Namespace: "ns2"}},
	}

	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Expected %+v, got %+v", expected, res)
	}
}

func TestGetNamespaces(t *testing.T) {
	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	createTestNamespaces(client, "x1", "x2")
	res, err := client.GetObjects("namespace")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := []KubeObject{
		{TypeMeta: TypeMeta{Kind: "namespace"}, ObjectMeta: ObjectMeta{Name: "x1"}},
		{TypeMeta: TypeMeta{Kind: "namespace"}, ObjectMeta: ObjectMeta{Name: "x2"}},
	}

	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Expected %+v, got %+v", expected, res)
	}
}

func TestGetDeployments(t *testing.T) {
	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	deploys := [][]string{
		{"ns1","deploy1"},
		{"ns1","deploy2"},
		{"ns2","deploy3"},
	}
	createTestDeployments(client, deploys)
	res, err := client.GetObjects("deployment")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := []KubeObject{
		{TypeMeta: TypeMeta{Kind: "deployment"}, ObjectMeta: ObjectMeta{Name: "deploy1", Namespace: "ns1"}},
		{TypeMeta: TypeMeta{Kind: "deployment"}, ObjectMeta: ObjectMeta{Name: "deploy2", Namespace: "ns1"}},
		{TypeMeta: TypeMeta{Kind: "deployment"}, ObjectMeta: ObjectMeta{Name: "deploy3", Namespace: "ns2"}},
	}

	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Expected %+v, got %+v", expected, res)
	}
}

func TestGetServices(t *testing.T) {
	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	svcs := [][]string{
		{"ns1","svc1"},
		{"ns1","svc2"},
		{"ns2","svc3"},
	}
	createTestSvcs(client, svcs)
	res, err := client.GetObjects("service")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := []KubeObject{
		{TypeMeta: TypeMeta{Kind:"service"}, ObjectMeta: ObjectMeta{Name: "svc1", Namespace: "ns1"}},
		{TypeMeta: TypeMeta{Kind:"service"}, ObjectMeta: ObjectMeta{Name: "svc2", Namespace: "ns1"}},
		{TypeMeta: TypeMeta{Kind:"service"}, ObjectMeta: ObjectMeta{Name: "svc3", Namespace: "ns2"}},
	}

	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Expected %+v, got %+v", expected, res)
	}
}

func TestGetNodes(t *testing.T) {
	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	createTestNodes(client, "x1", "x2")
	res, err := client.GetObjects("node")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := []KubeObject{
		{TypeMeta: TypeMeta{Kind:"node"}, ObjectMeta: ObjectMeta{Name: "x1"}},
		{TypeMeta: TypeMeta{Kind:"node"}, ObjectMeta: ObjectMeta{Name: "x2"}},
	}

	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Expected %+v, got %+v", expected, res)
	}
}

func TestPing(t *testing.T) {
	//setup()
	//defer teardown()
	//
	//mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprint(w, `OK`)
	//},
	//)
	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}
	createTestNodes(client, "x1")
	err := client.Ping()
	assert.NoError(t, err)

}

func TestPingError(t *testing.T) {
	//setup()
	//defer teardown()
	//
	//mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	http.Error(w, "Test error", 400)
	//},
	//)
	client := &DefaultKubeClient{
		kcClient: testclient.NewSimpleClientset(),
	}

	err := client.Ping()
	assert.Error(t, err)
}

func createTestNodes(client *DefaultKubeClient, names ...string) {
	for _, name := range names {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		client.kcClient.CoreV1().Nodes().Create(node)
	}
}

func createTestNamespaces(client *DefaultKubeClient, names ...string) {
	for _, name := range names {
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		client.kcClient.CoreV1().Namespaces().Create(ns)
	}
}

func createTestCMs(client *DefaultKubeClient, names [][]string) {
	for _, cmsInNS := range names {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: cmsInNS[1],
			},
		}
		client.kcClient.CoreV1().ConfigMaps(cmsInNS[0]).Create(cm)
	}
}

func createTestSvcs(client *DefaultKubeClient, names [][]string) {
	for _, svcs := range names {
		sv := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcs[1],
			},
		}
		client.kcClient.CoreV1().Services(svcs[0]).Create(sv)
	}
}

func createTestDeployments(client *DefaultKubeClient, names [][]string){
	for _, deploys := range names {
		d := &v1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: deploys[1],
			},
		}
		client.kcClient.ExtensionsV1beta1().Deployments(deploys[0]).Create(d)
	}
}