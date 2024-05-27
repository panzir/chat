package main

import (
    "context"
    "log"
    "strings"
    "strconv"
    "time"
    amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
    if err != nil {
        log.Panicf("%s: %s", msg, err)
    }
}

func (app *Application) RabbitPublish(msg string) {
    conn, err := amqp.Dial("amqp://guest:guest@" + HOST_RABBIT)
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    err = ch.ExchangeDeclare(
        "chat",   // name
        "fanout", // type
        true,     // durable
        false,    // auto-deleted
        false,    // internal
        false,    // no-wait
        nil,      // arguments
    )
    failOnError(err, "Failed to declare an exchange")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    body := msg
    err = ch.PublishWithContext(ctx,
        "chat", // exchange
        "",     // routing key
        false,  // mandatory
        false,  // immediate
        amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte(body),
        })
    failOnError(err, "Failed to publish a message")

    //log.Printf(" [x] Sent %s", body)
} 

func (app *Application) RabbitReceiver() {
    conn, err := amqp.Dial("amqp://guest:guest@" + HOST_RABBIT)
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    err = ch.ExchangeDeclare(
        "chat",   // name
        "fanout", // type
        true,     // durable
        false,    // auto-deleted
        false,    // internal
        false,    // no-wait
        nil,      // arguments
    )
    failOnError(err, "Failed to declare an exchange")

    q, err := ch.QueueDeclare(
        "",    // name
        false, // durable
        false, // delete when unused
        true,  // exclusive
        false, // no-wait
        nil,   // arguments
    )
    failOnError(err, "Failed to declare a queue")

    err = ch.QueueBind(
        q.Name, // queue name
        "",     // routing key
        "chat", // exchange
        false,
        nil,
    )
    failOnError(err, "Failed to bind a queue")

    msgs, err := ch.Consume(
        q.Name, // queue
        "",     // consumer
        true,   // auto-ack
        false,  // exclusive
        false,  // no-local
        false,  // no-wait
        nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    var forever chan struct{}
    
    go func() {
        for d := range msgs {
            s := string(d.Body)
            ss := strings.Split(s,";")
            id, _ := strconv.Atoi(ss[0])
            room, _ := strconv.Atoi(ss[1])
            header := MessageHeader{Id: id, Room: room}
            
            app.messagePool <- header
            
            //log.Printf(" [x] %s", d.Body)
        }
    }()

    log.Printf(" [*] Waiting for chat. To exit press CTRL+C")
    <-forever
}
