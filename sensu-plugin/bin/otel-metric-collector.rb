#!/usr/bin/env ruby
#
# Collects OpenTelemetry metrics from a collector Prometheus exporter
# and converts to Carbon format for the MID server to process.
#
# Notes:
#   - The script will output metrics in Carbon format to STDOUT
#   - The script will exit with a status code of 0 upon success
#   - The script will exit with a status code of 2 if there is a failure
#   - The script will output a message to STDERR if there is a failure
#

require 'sensu-plugin/check/cli'
require 'net/http'

DEFAULT_HOST = 'localhost'
DEFAULT_PORT = 9090
DEFAULT_TIMEOUT = 10
DEFAULT_PATH = '/metrics'
PROCESS_NAME = 'otelcol-servicenow'
LAUNCH_COLLECTOR_CMD = "HOSTNAME=#{Socket.gethostname} ./#{PROCESS_NAME} --config config.yaml &"

class CollectOTelMetrics < Sensu::Plugin::Check::CLI
  option :timeout,
    long: '--timeout TIMEOUT',
    proc: proc(&:to_f),
    default: DEFAULT_TIMEOUT

  option :host,
    long: '--host HOST',
    default: DEFAULT_HOST

  option :port,
    long: '--port PORT',
    default: DEFAULT_PORT

    def process_running?()
      `ps aux | grep -v grep | grep #{PROCESS_NAME}`.lines.count > 0
    end

  # Convert Prometheus format to Carbon format
  def prometheus_to_carbon(prometheus_metric)
    metric, value, timestamp = prometheus_metric.split(' ')

    metric_name, labels_string = metric.split('{')
    labels_string.chomp!('}') if labels_string

    labels = labels_string ? labels_string.split(',').map { |label| label.split('=') }.to_h : {}

    carbon_metric_name = metric_name.gsub('_', '.').sub('.', '-')
    carbon_labels = labels.map { |k, v| "#{k}=#{v.gsub('"', '')}" }.join(';')
    carbon_metric_name += ";#{carbon_labels}" unless carbon_labels.empty?

    timestamp = (timestamp.to_i / 1000).to_s

    carbon_metric = "#{carbon_metric_name} #{value} #{timestamp}"

    carbon_metric
  end

  # Fetches metric from prometheus endpoint passed in as CLI options
  def fetch_and_process_metrics()
    uri = URI("http://#{config[:host]}:#{config[:port]}#{DEFAULT_PATH}")
    response = Net::HTTP.get_response(uri)

    if response.is_a?(Net::HTTPSuccess)
      response.body.each_line do |line|
        line.strip!
        next if line.empty? || line.start_with?('#')

        carbon_metric = prometheus_to_carbon(line)
        puts carbon_metric
      end
    else
      critical "Failed to fetch metrics: #{response.message}"
      exit 1
    end
  end

  def run
    if !process_running?
      output = %x{#{LAUNCH_COLLECTOR_CMD} 2>&1} # Capture both stdout and stderr
      status = $?.exitstatus
      if status != 0
        puts output
        puts status
        critical "Failed to launch collector: #{output}"
        exit 1
      end

      sleep 15
    end

    fetch_and_process_metrics
    exit 0
  end
end
