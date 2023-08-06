function init() {

    const socket = new WebSocket("ws://localhost:8888/helo");

    // Connection opened
    socket.onopen = () => {
        var c = {
            id: crypto.randomUUID(),
            cmd: 6,
            data: ["test"]
        }
        const msg = JSON.stringify(c)
        socket.send(msg)
        console.log("> ", msg)
    };

    // Listen for messages
    socket.onmessage = (ev) => {
        console.log("< ", ev.data);
    };

    socket.onclose = (ev) => {
        console.log("disconnected")
    }
}

init();
