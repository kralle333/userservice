
create-topics:
	kafkactl create topic userservice.user.added
	kafkactl create topic userservice.user.removed

create-mongodb-indexes:
	mongosh --eval 'use userservice; db.kafkaoutbox.createIndex({ id: 1 })'
	mongosh --eval 'use userservice; db.user.createIndex({ msg_id: 1 })'
	mongosh --eval 'use functional; db.kafkaoutbox.createIndex({ id: 1 })'
	mongosh --eval 'use functional; db.user.createIndex({ msg_id: 1 })'


gen-proto: gen-grpc gen-kafka-schemas

clean-proto:
	find proto/. -name "*.pb.go" -type f -delete
gen-grpc:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative  proto/grpc/*.proto

gen-kafka-schemas:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative  proto/kafkaschema/*.proto