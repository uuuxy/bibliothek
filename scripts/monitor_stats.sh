#!/bin/bash

# Log file
LOG_FILE="docker_stats.log"

# Clear or create log file
echo "Monitoring started at $(date)" > $LOG_FILE

# Run for approx 6 hours (24 iterations of 15 minutes = 360 mins, plus 1 to cover the end)
for i in {1..25}; do
  echo "--- $(date) ---" >> $LOG_FILE
  docker stats --no-stream bibliothek-backend-local postgres-db >> $LOG_FILE
  
  if [ $i -lt 25 ]; then
    sleep 900 # 15 minutes
  fi
done

echo "Monitoring finished at $(date)" >> $LOG_FILE
