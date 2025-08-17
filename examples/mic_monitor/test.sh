#!/bin/bash

# Quick test of mic monitor functionality
# This script tests basic commands automatically

echo "Testing MacAudio Mic Monitor..."

# Run the mic monitor with automated input
(
  sleep 2         # Give it time to start
  echo "status"   # Check current status
  sleep 1
  echo "i 50"     # Set input to 50%
  sleep 1  
  echo "m 20"     # Set master to 20%
  sleep 1
  echo "status"   # Check status again
  sleep 1
  echo "mute"     # Toggle mute
  sleep 1
  echo "mute"     # Toggle back
  sleep 1
  echo "quit"     # Exit cleanly
) | ./mic_monitor

echo "Test complete!"
