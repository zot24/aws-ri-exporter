package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// BadInstancesInput bad instances input that could not be and therefore fail when parsing
const BadInstancesInput = "Instaces input could not be parsed"

var (
	metricsAddr = flag.String("metrics-address", ":9900", "The address to listen on for Prometheus metrics requests.")
	namespace   = flag.String("namespace", "cloud", "The namespace to use in Prometheus to store these metrics")

	ec2Svc *ec2.EC2

	normalizedFactors = map[string]float32{
		"nano":     0.25,
		"micro":    0.5,
		"small":    1,
		"medium":   2,
		"large":    4,
		"xlarge":   8,
		"2xlarge":  16,
		"4xlarge":  32,
		"8xlarge":  64,
		"9xlarge":  72,
		"10xlarge": 80,
		"12xlarge": 96,
		"16xlarge": 128,
		"18xlarge": 144,
		"24xlarge": 192,
		"32xlarge": 256,
	}

	normalizeMinimum = map[string]string{
		// General purpose
		"t2":  "nano",
		"t3":  "nano",
		"m3":  "medium",
		"m4":  "large",
		"m5":  "large",
		"m5d": "large",
		// Compute optimized
		"c3":  "large",
		"c4":  "large",
		"c5":  "large",
		"c5d": "large",
		// Memory optimized
		"r3":  "large",
		"r4":  "large",
		"r5":  "large",
		"r5d": "large",
		// Storage optimized,
		"i2": "xlarge",
		"i3": "large",
	}

	instancesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: *namespace,
			Subsystem: "aws_compute_ec2_ri",
			Name:      "instances",
			Help:      "A gauge vector of instances",
		},
		[]string{
			"type",
			"size",
		},
	)

	reserveInstancesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: *namespace,
			Subsystem: "aws_compute_ec2_ri",
			Name:      "reserved_instances",
			Help:      "A gauge vector of reserved instances",
		},
		[]string{
			"type",
			"size",
		},
	)

	normalizedInstancesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: *namespace,
			Subsystem: "aws_compute_ec2_ri",
			Name:      "normalized_instances",
			Help:      "A gauge vector of normalized instances",
		},
		[]string{
			"type",
		},
	)

	normalizedReserveInstancesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: *namespace,
			Subsystem: "aws_compute_ec2_ri",
			Name:      "normalized_reserve_instances",
			Help:      "A gauge vector of normalized reserve instances",
		},
		[]string{
			"type",
		},
	)
)

func init() {
	prometheus.MustRegister(instancesGauge)
	prometheus.MustRegister(reserveInstancesGauge)
	prometheus.MustRegister(normalizedInstancesGauge)
	prometheus.MustRegister(normalizedReserveInstancesGauge)

	// load session from shared config and create new EC2 client
	ec2Svc = ec2.New(session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})))
}

func main() {
	flag.Parse()

	log.Printf("Listening on %s", *metricsAddr)
	http.Handle("/metrics", metricsHandler(promhttp.Handler()))
	log.Fatal(http.ListenAndServe(*metricsAddr, nil))
}

func metricsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := metrics()
		if err != nil {
			log.Println(err)
		}
		h.ServeHTTP(w, r)
	})
}

func metrics() error {
	instances := getInstances()
	reserveInstances := getReserveInstances()

	log.Print("instances ", instances)
	log.Print("reserve_instances ", reserveInstances)

	instancesGauge.Reset()
	for k, v := range instances {
		parts := strings.Split(k, ".")
		instancesGauge.With(prometheus.Labels{"type": parts[0], "size": parts[1]}).Set(float64(v))
	}

	reserveInstancesGauge.Reset()
	for k, v := range reserveInstances {
		parts := strings.Split(k, ".")
		reserveInstancesGauge.With(prometheus.Labels{"type": parts[0], "size": parts[1]}).Set(float64(v))
	}

	normalizedInstances, err := normalizeInstances(instances) // float32
	if err != nil {
		return err
	}

	normalizedReserveInstances, err := normalizeInstances(reserveInstances) // float32
	if err != nil {
		return err
	}

	log.Print("normalized_instances ", normalizedInstances)
	log.Print("normalized_reserve_instances ", normalizedReserveInstances)

	normalizedInstancesGauge.Reset()
	for k, v := range normalizedInstances {
		parts := strings.Split(k, ".")
		normalizedInstancesGauge.With(prometheus.Labels{"type": parts[0]}).Set(float64(v))
	}

	normalizedReserveInstancesGauge.Reset()
	for k, v := range normalizedReserveInstances {
		parts := strings.Split(k, ".")
		normalizedReserveInstancesGauge.With(prometheus.Labels{"type": parts[0]}).Set(float64(v))
	}

	return nil
}

func getInstances() map[string]int64 {
	instances := map[string]int64{}

	// create a input with the following filters for our ec2 instances (running and pending state name)
	ec2Input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	}

	// call to get detailed information on each instance
	ec2Instances, err := ec2Svc.DescribeInstances(ec2Input)
	if err != nil {
		log.Fatal(err)
	} else {
		for _, r := range ec2Instances.Reservations {
			for _, i := range r.Instances {
				instances[*i.InstanceType]++
			}
		}
	}
	return instances
}

func getReserveInstances() map[string]int64 {
	reserveInstances := map[string]int64{}

	// create a input with the following filters for our ec2 instances (running and pending state name)
	reserveInstancesInput := &ec2.DescribeReservedInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("state"),
				Values: []*string{
					aws.String("active"),
				},
			},
		},
	}

	// call to get detailed information on each reserve instance
	ec2reserveInstances, err := ec2Svc.DescribeReservedInstances(reserveInstancesInput)
	if err != nil {
		log.Fatal(err)
	} else {
		for _, r := range ec2reserveInstances.ReservedInstances {
			reserveInstances[*r.InstanceType] += *r.InstanceCount
		}
	}

	return reserveInstances
}

func normalizeInstances(instances map[string]int64) (map[string]float32, error) {
	normalizedInstances := map[string]float32{}

	for k, v := range instances {
		parts := strings.Split(k, ".")

		if len(parts) < 2 {
			return normalizedInstances, errors.New(BadInstancesInput)
		}

		instanceType := parts[0]
		instanceSize := parts[1]
		minimumSize := normalizeMinimum[instanceType]

		if minimumSize == "" {
			minimumSize = instanceSize
		}

		normalizedInstances[instanceType] += normalizedFactors[instanceSize] / normalizedFactors[minimumSize] * float32(v)
	}

	return normalizedInstances, nil
}
