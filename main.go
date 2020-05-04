package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"context"

	// Attempt 1
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// // Controller
	// "k8s.io/klog"
	// "k8s.io/apimachinery/pkg/fields"
	// "k8s.io/apimachinery/pkg/util/runtime"
	// "k8s.io/apimachinery/pkg/util/wait"
	// "k8s.io/client-go/tools/cache"
	// "k8s.io/client-go/util/workqueue"
	// "time"

	// Custom module
	"github.com/PreritO/Deploy-DC/podWatcher"
)

func configK8() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func deployer(deploymentPath string, namespace string, clientset *kubernetes.Clientset) ([]string, error) {
	// Now, we need to extract all pod names from the files in the deployment path so that we can keep track of them

	fmt.Printf("Reading Directory: %s\n", deploymentPath)
	files, err := ioutil.ReadDir(deploymentPath)
	if err != nil {
		fmt.Printf("[ERROR] Can't read files from deployment path directory: %s\n", deploymentPath)
		fmt.Println(err)
		os.Exit(1)
	}
	//acceptedK8sTypes := regexp.MustCompile(`(Deployment)`)
	podNames := []string{}
	// Todo:  get the namespace of the application here

	decode := scheme.Codecs.UniversalDeserializer().Decode
	for _, item := range files {
		filePath := fmt.Sprintf("%v", deploymentPath) + item.Name()
		fmt.Println("Reading File: %s", filePath)

		// Attempt 1
		yamlFile, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error in reading file: %s, Error: %s\n", filePath, err)
		}
		// There can be multiple yaml definitions per file
		docs := strings.Split(string(yamlFile), "\n---")
		res := []byte{}
		// Trim whitespace in both ends of each yaml docs.
		for _, doc := range docs {
			content := strings.TrimSpace(doc)
			// Ignore empty docs
			if content != "" {
				res = append(res, content+"\n"...)
			}
			obj, groupVersionKind, err := decode(res, nil, nil)
			if err != nil {
				fmt.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
				continue
			}
			
			switch groupVersionKind.Kind {
			case "Deployment":
				fmt.Printf("Found Deployment! \n")
				deploymentsClient := clientset.AppsV1().Deployments(namespace)
				originalDeployment := obj.(*appsv1.Deployment)
				for i := 0; i < len(originalDeployment.Spec.Template.Spec.Containers); i++ {
					fmt.Printf("Container Name: %s\n", originalDeployment.Spec.Template.Spec.Containers[i].Name)
					podNames = append(podNames, originalDeployment.Spec.Template.Spec.Containers[i].Name)

					// Todo:  Update limits here of each individual container here to be total limit/total containers...
					// resource: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/resourcequota/resource_quota_controller_test.go
					// originalDeployment.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("100m")
					// originalDeployment.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceMemory] = resource.MustParse("1Gi")
					// fmt.Printf("Type: %++v\n", reflect.TypeOf(originalDeployment))
					// fmt.Printf("Container Limits: %s\n", originalDeployment.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceCPU])
				}
				// example resource: https://github.com/kubernetes/client-go/blob/master/examples/create-update-delete-deployment/main.go
				// API: https://godoc.org/k8s.io/api/apps/v1
				result, err := deploymentsClient.Create(context.TODO(), originalDeployment, metav1.CreateOptions{})
				if err != nil {
					panic(err)
				}
				fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
			case "Service":
				fmt.Printf("Found Service! \n")
				servicesClientInterface := clientset.CoreV1().Services(namespace)
				originalService := obj.(*corev1.Service)
				result, err := servicesClientInterface.Create(context.TODO(), originalService, metav1.CreateOptions{})
				if err != nil {
					panic(err)
				}
				fmt.Printf("Created Service %q.\n", result.GetObjectMeta().GetName())
			default:
				fmt.Printf("Unsupported Type: %s \n", groupVersionKind.Kind)
				continue
			}
		}
		fmt.Printf("\n")
	}
	return podNames, nil
}

func main() {
	// First, we parse the application definition file for app statisitics
	appDefFilePtr := flag.String("f", "", "App Definition File to parse. (Required)")
	flag.Parse()

	if *appDefFilePtr == "" {
		fmt.Println("Must pass in a file path to the app definition file: ")
		flag.PrintDefaults()
		os.Exit(1)
	}

	jsonAppDefFile, err := os.Open(*appDefFilePtr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("[DBG] Successfully opened %s\n", *appDefFilePtr)
	defer jsonAppDefFile.Close()

	jsonByteValue, _ := ioutil.ReadAll(jsonAppDefFile)
	var result map[string]interface{}
	json.Unmarshal([]byte(jsonByteValue), &result)

	appName := result["name"]
	agentIPs := result["agentIPs"]
	gcmIP := result["gcmIP"]
	deploymentPath := result["deploymentPath"]
	namespace := result["namespace"]

	if deploymentPath == nil {
		fmt.Printf("[ERROR] Application Deployment spath is null: \n")
		os.Exit(1)
	}

	fmt.Printf("AppName: %s, agent IP: %s, gcmIP: %s, deploymentPath: %s \n", appName, agentIPs, gcmIP, deploymentPath)

	// Now, we configure the K8s ClientSet and get a reference to that
	fmt.Printf("[DBG] Configuring K8s ClientSet\n")
	clientset := configK8()

	// Add a Pod watcher/listener here for pods added to the appropriate namespace
	fmt.Printf("[DBG] Adding a Pod watcher for namespace: %s\n", namespace.(string))
	//podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", namespace.(string) , fields.Everything())
	podListWatcher := podWatcher.ListWatcher(namespace.(string), clientset)
	//queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	queue := podWatcher.CreateQueue()

	controller := podWatcher.SetupWatcher(podListWatcher, queue)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)


	// Deploy the Application nominally - as it would be via `kubectl apply -f` and get the container names of all pods in the application
	fmt.Printf("[DBG] Deploying Application and Gathering List of Active Pods.. \n")
	podList, err := deployer(deploymentPath.(string), namespace.(string) ,clientset)
	if err != nil {
		fmt.Printf("Error in parsing through deployment")
	}
	fmt.Printf("Deployed Application Pod Names: %s\n", podList)

	select {}

}
