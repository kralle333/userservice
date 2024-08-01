# Golang User Service
Simple backend service for handling users, features:
- add/remove/update/list users
- uses grpc for handling requests
- event raising using kafka, using proto for schemas
- storing user data using mongodb
- easy to add new healthchecks for new services
- architecture supports changing DB layer or API layer
- outbox pattern used for improving data consistency
- generation of config schemas
- comprehensive testing

