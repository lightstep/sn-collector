#!/usr/bin/env ruby

# This is just a simple TCP server that proxies metrics output from the
# carbonreceiver to stdout so it can be read by a sensu metrics plugin.

require 'socket'
require 'timeout'

PORT = 2003
HOST = '0.0.0.0'
TIMEOUT = 30

# TODO: run the collector binary / check if it's running.

def start_server
  server = TCPServer.new(HOST, PORT)
  #puts "Server started on #{HOST}:#{PORT}"

  begin
    Timeout.timeout(TIMEOUT) do
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

start_server