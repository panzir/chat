package main

import (
    "context"
    "log"
    "time"
    "encoding/json"
    amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
    if err != nil {
        log.Panicf("%s: %s", msg, err)
    }
}

func (app *Application) RabbitPublish(jsn []byte) {
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

    body := jsn
    err = ch.PublishWithContext(ctx,
        "chat", // exchange
        "",     // routing key
        false,  // mandatory
        false,  // immediate
        amqp.Publishing{
            //ContentType: "text/plain",
            ContentType: "application/json",
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
            header := MessageHeader{}
            _ = json.Unmarshal(d.Body,&header)
            app.messagePool <- header
            
            //log.Printf(" [x] %s", d.Body)
        }
    }()

    log.Printf(" [*] Waiting for chat. To exit press CTRL+C")
    <-forever
}
