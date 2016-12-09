package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"flag"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/zalando-incubator/kube-ingress-aws-controller/aws"
	"github.com/zalando-incubator/kube-ingress-aws-controller/kubernetes"
)

var (
	apiServerBaseURL string
	pollingInterval  time.Duration
)

func waitForTerminationSignals(signals ...os.Signal) chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	return c
}

func loadEnviroment() error {
	flag.Usage = usage
	flag.StringVar(&apiServerBaseURL, "api-server-base-url", "http://127.0.0.1:8001", "sets the kubernetes api server base url. "+
		"if empty will try to use the common proxy url http://127.0.0.1:8001")
	flag.DurationVar(&pollingInterval, "polling-interval", 30*time.Second, "sets the polling interval for ingress resources. "+
		"The flag accepts a value acceptable to time.ParseDuration. Defaults to 30 seconds")
	flag.Parse()

	if tmp, defined := os.LookupEnv("API_SERVER_BASE_URL"); defined {
		apiServerBaseURL = tmp
	}

	if tmp, defined := os.LookupEnv("POLLING_INTERVAL"); defined {
		interval, err := time.ParseDuration(tmp)
		if err != nil || interval <= 0 {
			return err
		}
		pollingInterval = interval
	}

	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "where options can be:")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.Printf("starting %s", os.Args[0])
	var err error
	if err = loadEnviroment(); err != nil {
		log.Fatal(err)
	}

	session := session.Must(session.NewSession())
	awsAdapter, err := aws.NewAdapter(session)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("controller manifest:")
	log.Printf("\tKubernetes API server: %s\n", apiServerBaseURL)
	log.Printf("\tcurrent vpc id: %s\n", awsAdapter.VpcID())
	log.Printf("\tcurrent instance id: %s\n", awsAdapter.InstanceID())
	log.Printf("\tsecurity group id: %s\n", awsAdapter.SecurityGroupID())
	log.Printf("\ttarget group ARN: %s\n", awsAdapter.TargetGroupARN())
	log.Printf("\tprivate subnet ids: %s\n", awsAdapter.PrivateSubnetIDs())
	log.Printf("\tpublic subnet ids: %s\n", awsAdapter.PublicSubnetIDs())

	kubernetesClient := kubernetes.NewClient(apiServerBaseURL)
	go startPolling(awsAdapter, kubernetesClient, pollingInterval)
	<-waitForTerminationSignals(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Printf("terminating %s\n", os.Args[0])
}
