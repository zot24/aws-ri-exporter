{
  _config+:: {
    aws_ri_selector: 'job=~".*aws-ri-exporter.*"',
  },
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'aws-compute-ec2-ri',
        rules: [
          {
            alert: 'AWSRIUtilizationUnder100Percent',
            expr: '((cloud_aws_compute_ec2_ri_normalized_instances / cloud_aws_compute_ec2_ri_normalized_reserve_instances) * 100) < 100',
            'for': '10m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Reserve Instances of type "{{ $labels.type }}": under 100% utilization ({{ $value }}).',
              description: 'Check that the utilization of our AWS Reserve Instances its 100% all the time',
            },
          },
        ],
      },
    ],
  },
  grafanaDashboards+:: {
    'aws-ri-utilization.json': (import 'handcrafted/aws-ri-utilization.json'),
  },
}
