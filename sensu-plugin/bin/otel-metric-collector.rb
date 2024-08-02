#!/usr/bin/env ruby

require 'sensu-plugin/check/cli'
require 'json'
require 'socket'
require 'timeout'

DEFAULT_PORT = 2003
DEFAULT_HOST = '0.0.0.0'
DEFAULT_TIMEOUT = 30

#
# Collects OpenTelemetry metrics received from a carbon exporter
# listening on a TCP port.
#
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

  def start_server
    server = TCPServer.new(config[:host], config[:port])
    #puts "Server started on #{HOST}:#{PORT}"

    begin
      Timeout.timeout(config[:timeout]) do
        loop do
          client = server.accept
          Thread.new(client) do |client_connection|
            handle_client(client_connection)
          end
        end
      end
    rescue Timeout::Error
      #puts "Server timeout reached. Shutting down..."
    ensure
      # TODO: handle close nicely to avoid
      # WARNING: Check did not exit! You should call an exit code method.
      server.close
    end
  end

  def handle_client(client)
    loop do
      line = client.gets
      break if line.nil? || line.chomp.empty?

      puts line.chomp
      client.puts line
    end

    client.close
  end

  def run
    start_server
  end
end
