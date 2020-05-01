package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	// Attempt 1
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	//corev1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/api/resource"
	//"reflect"
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

func deployer(filePath *string) ([]string, int) {
	jsonAppDefFile, err := os.Open(*filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("[DBG] Successfully opened %s\n", *filePath)
	defer jsonAppDefFile.Close()

	jsonByteValue, _ := ioutil.ReadAll(jsonAppDefFile)
	var result map[string]interface{}
	json.Unmarshal([]byte(jsonByteValue), &result)

	appName := result["name"]
	agentIPs := result["agentIPs"]
	gcmIP := result["gcmIP"]
	deploymentPath := result["deploymentPath"]

	if deploymentPath == nil {
		fmt.Printf("[ERROR] Application Deployment path is null: \n")
		os.Exit(1)
	}

	fmt.Printf("AppName: %s, agent IP: %s, gcmIP: %s, deploymentPath: %s \n", appName, agentIPs, gcmIP, deploymentPath)

	//clientset := configK8()

	// Now, we need to extract all pod names from the files in the deployment path so that we can keep track of them

	fmt.Printf("Reading Directory: %s\n", deploymentPath)
	files, err := ioutil.ReadDir(deploymentPath.(string))
	if err != nil {
		fmt.Printf("[ERROR] Can't read files from deployment path directory: %s\n", deploymentPath)
		fmt.Println(err)
		os.Exit(1)
	}
	//acceptedK8sTypes := regexp.MustCompile(`(Deployment)`)
	podNames := []string{}

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
				// Here, we can parse through the deployment object
				originalDeployment := obj.(*appsv1.Deployment)
				for i := 0; i < len(originalDeployment.Spec.Template.Spec.Containers); i++ {
					fmt.Printf("Container Name: %s\n", originalDeployment.Spec.Template.Spec.Containers[i].Name)
					podNames = append(podNames, originalDeployment.Spec.Template.Spec.Containers[i].Name)

					// Todo:  Update limits here and then deploy the deployment
					// originalDeployment.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("100m")
					// originalDeployment.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceMemory] = resource.MustParse("1Gi")
					// fmt.Printf("Type: %++v\n", reflect.TypeOf(originalDeployment))
					// fmt.Printf("Container Limits: %s\n", originalDeployment.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceCPU])
				}
				
			default:
				fmt.Printf("Unsupported Type: %s \n", groupVersionKind.Kind)
				continue
			}
		}
		fmt.Printf("\n")
	}
	return podNames, 0
}

func main() {
	// Replaces main.cpp in that it parses the fields from the passed in JSON app_definition file:
	appDefFilePtr := flag.String("f", "", "App Definition File to parse. (Required)")
	flag.Parse()

	if *appDefFilePtr == "" {
		fmt.Println("Must pass in a file path to the app definition file: ")
		flag.PrintDefaults()
		os.Exit(1)
	}

	podList, err := deployer(appDefFilePtr)
	if err != 0 {
		fmt.Printf("Error in parsing through deployment")
	}
	fmt.Printf("All Pod Names: %s\n", podList)

}
