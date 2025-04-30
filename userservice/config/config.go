package config

const HTTPPort = ":3000"
const RedisAddr = "redis:6379"
const UserDBPath = "users.db"
const JWTSecret = "secret-key"
const GRPCAddress = "notificationservice:50051"
const MAXReqPerUser = 100
const KafkaBrokers = "kafka:9093"

// const KafkaBrokers = "localhost:9092"
const EmailTopic = "email"
const LogsTopic = "logs"
const MessageTopic = "message"
