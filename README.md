# Golang User service
Simple backend service for handling users, features:
- add/remove/update/list users
- uses grpc for handling requests
- event raising using kafka
- storing user data using mongodb
- easy to add new healthchecks for new services
- architecture supports changing DB layer or API layer
- outbox pattern used for improving data consistency
- comprehensive testing

