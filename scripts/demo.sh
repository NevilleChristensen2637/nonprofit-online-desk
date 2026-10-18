#!/bin/sh
set -eu

curl --fail-with-body --request POST http://localhost:8080/brief \
  --header 'Content-Type: application/json' \
  --data '{"channel":"nonprofit-operations","assignments":[{"client_id":"staff-ana","name":"Ana","desk":"donor_receipts","pending":4},{"client_id":"staff-mo","name":"Mo","desk":"volunteer_reminders","pending":7},{"client_id":"staff-lee","name":"Lee","desk":"campaign_reporting","pending":1}]}'
