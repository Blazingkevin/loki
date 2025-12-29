#!/bin/bash

# Quick Demo of Loki Chaos Engineering
# Shows real-time chaos in action

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║          Loki Chaos Engineering - Live Demo                 ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "This demo will:"
echo "  1. Start Loki with chaos config"
echo "  2. Make 15 requests showing chaos in action"
echo "  3. Display results with timing and status codes"
echo ""
echo "Press Enter to start..."
read

# Start server
echo "🚀 Starting Loki server with chaos..."
./bin/loki serve test/petstore.yaml \
  --chaos test/test-chaos-config.yaml \
  --port 9999 \
  --log-level info > /tmp/loki-demo.log 2>&1 &

SERVER_PID=$!
sleep 2

if ! ps -p $SERVER_PID > /dev/null; then
  echo "❌ Server failed to start"
  exit 1
fi

echo "✅ Server running on http://localhost:9999"
echo ""
echo "Making 15 requests to /pets..."
echo "Watch for:"
echo "  • ⚡ Fast responses (~20-30ms)"  
echo "  • 🐌 Slow responses (50-200ms latency injection)"
echo "  • ❌ Error responses (500/503 error injection)"
echo ""
echo "═══════════════════════════════════════════════════════════════"

# Make requests and show results
for i in {1..15}; do
  echo -n "Request $i: "
  
  start=$(python3 -c 'import time; print(int(time.time() * 1000))')
  response=$(curl -s -w "\n%{http_code}" -o /tmp/response.json http://localhost:9999/pets 2>/dev/null)
  end=$(python3 -c 'import time; print(int(time.time() * 1000))')
  
  http_code=$(echo "$response" | tail -n1)
  duration=$((end - start))
  
  # Color code based on result
  if [ "$http_code" = "200" ]; then
    if [ $duration -gt 45 ]; then
      echo "🐌 HTTP $http_code - ${duration}ms (LATENCY CHAOS)"
    else
      echo "⚡ HTTP $http_code - ${duration}ms (normal)"
    fi
  else
    echo "❌ HTTP $http_code - ${duration}ms (ERROR CHAOS)"
    # Show error message
    error_msg=$(cat /tmp/response.json | python3 -c "import sys, json; print(json.load(sys.stdin).get('error', ''))" 2>/dev/null)
    echo "   └─ Error: $error_msg"
  fi
  
  sleep 0.3
done

echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "📊 Statistics:"
echo ""

# Analyze logs
chaos_count=$(grep -c "chaos applied" /tmp/loki-demo.log 2>/dev/null || echo "0")
latency_count=$(grep "chaos applied" /tmp/loki-demo.log | grep -c "latency" 2>/dev/null || echo "0")
error_count=$(grep "chaos applied" /tmp/loki-demo.log | grep -c "error" 2>/dev/null || echo "0")

echo "  Total chaos events: $chaos_count"
echo "  Latency injections: $latency_count"
echo "  Error injections: $error_count"
echo ""

# Show sample chaos log
echo "📝 Sample Chaos Log Entry:"
sample=$(grep "chaos applied" /tmp/loki-demo.log | head -n1)
if [ -n "$sample" ]; then
  echo "$sample" | python3 -m json.tool 2>/dev/null | head -n 10
fi

echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null
rm -f /tmp/loki-demo.log /tmp/response.json

echo "✅ Demo complete!"
echo ""
echo "💡 Try it yourself:"
echo "   ./bin/loki serve test/petstore.yaml --chaos test/test-chaos-config.yaml"
echo ""
