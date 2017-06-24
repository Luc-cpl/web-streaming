var Greeting = React.createClass({
    render: function() {
    return (
        <div id="container">
        </div>
    )
    }
});

ReactDOM.render(
    <Greeting/>,
    document.getElementById('greeting-div')
);

var  ws = initWS();

function initWS() {
    var socket = new WebSocket("ws://localhost:8080/ws"),
        container = $("#container")
    socket.onopen = function() {
        container.append("<p>Socket is open</p>");
    };
    socket.onmessage = function (e) {
        container.html(e.data);
    }
    socket.onclose = function () {
        container.append("<p>Socket closed</p>");
    }
    return socket;
}