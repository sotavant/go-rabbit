package main

import (
	"github.com/streadway/amqp"
	"bitbucket.org/sotavant/rabbitOxpa/lib"
	"fmt"
)

const socketName = "/tmp/rabbit-oxpa.lock"

func main()  {
	// flock for one instance of program
	lib.Flock(socketName)

	conf, err := lib.Config()
	common := lib.Common{Config: conf.Common}
	common.FailOnError(err, "Failed load config")

	conn, err := amqp.Dial("amqp://" + conf.Rabbit.User + ":"  + conf.Rabbit.Pass + "@" + conf.Rabbit.Host + ":5672")
	common.FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	db := lib.Database{Config: conf.Database}
	task := lib.Task{Config: conf.Common, Database: db, Common: common}
	task.RecoverState()

	forever := make(chan bool)

	for j := 0; j < conf.Common.StreamCount; j++ {
		go RunChannel(conn, common, conf, &task, j)
	}

	<-forever
}

func RunChannel(conn *amqp.Connection, common lib.Common, conf lib.AppConfig, task *lib.Task, a int) {
	ch, err := conn.Channel()
	common.FailOnError(err, "Faild to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		conf.Rabbit.QueueName,
		true, // durable
		false,
		false,
		false,
		nil,
	)
	common.FailOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)

	msgs, err := ch.Consume(
		q.Name,
		"",
		false, // auto-ack
		false,
		false,
		false,
		nil,
	)
	common.FailOnError(err, "Failed in consuming errors")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			fmt.Printf("Channel %d: receiving message (%s)\n", a, d.Body)
			task.RunJob(d)
		}
	}()

	<- forever
}


