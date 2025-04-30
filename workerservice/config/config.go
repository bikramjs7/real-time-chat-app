package config

const (
	KafkaBrokers = "kafka:9093"
	//KafkaBrokers = "localhost:9092"
	KafkaGroupID = "chatapp-consumer-group"
	KafkaWorkers = 10
	GRPCAddress  = "notificationservice:50051"
	EmailTopic   = "email"
	LogsTopic    = "logs"
	MessageTopic = "message"

	SMTPHost     = "smtp.gmail.com"
	SMTPPort     = 587
	SMTPUser     = "bikram.7js@gmail.com"
	SMTPPassword = "nrnjsjgbtvoeniuj"
	FromEmail    = "bikram.7js@gmail.com"

	DatabasePath = "logs.db"
)
