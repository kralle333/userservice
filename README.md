# Disclaimer
This project is by no means done and there are tons of things that needs to be done for it to be production ready. 
Some of these could be:
- Telemetry data; tracing;logging
- Authentication/Authorization
- Create proper integration tests that do not have knowledge of implementation (e.g. raw http request instead of using grpc client)
- Add testing of repository layer
- MongoDB query logic cleanup

## Golang User Service
Simple backend service for handling user data.

### Features
- add/remove/update/list users
- uses grpc for handling requests
- event raising using kafka, using proto for schemas
- storing user data using mongodb
- easy to add new healthchecks for new services
- architecture supports changing DB layer or API layer
- outbox pattern used for improving data consistency
- generation of config schemas
- comprehensive testing

