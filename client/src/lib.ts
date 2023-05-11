export type EventArg = string | number | boolean | null | undefined | {[key: string]: EventArg} | EventArg[];
export type EventData = {[key: string]: EventArg};
export type EventHandler = (data: EventData) => void;

/**
 * The igo client is a wrapper for the default websocket client bringing compatibility with the igo server.
 */
export class IgoClient {

    private readonly _handlers: {[key: string]: EventHandler[]} = {};
    private readonly _reconnectTimeout: number;
    private readonly _url: string;
    private _socket: WebSocket | null = null;
    private _id: string = "";

    private _preConnectedHandler: (() => void) | null = null;
    private _connectedHandler: (() => void) | null = null;
    private _disconnectedHandler: (() => void) | null = null;

    /**
     * Constructs a new igo client and connects to the given url.
     * 
     * @param url The url to connect to.
     */
    constructor(url: string, reconnectTimeout: number = 5000) {
        this._url = url;
        this._reconnectTimeout = reconnectTimeout;
        this.connect();
    }

    /**
     * Emits a new event to the server.
     * 
     * @param event The event to emit.
     * @param data The data to send with the event.
     */
    public emit(event: string, data: EventData) {
        if (this._socket === null) {
            throw new Error("Socket is not connected");
        }
        this._socket.send(JSON.stringify({event, data}));
    }

    /**
     * Emits a new event to the server and awaits the server's response.
     * 
     * @param event The event to emit.
     * @param data The data to send with the event.
     * @returns A promise awaiting the server's response for the acknoledgement.
     */
    public emitWithAck(event: string, data: EventData): Promise<EventArg> {
        return new Promise((resolve, reject) => {
            if (this._socket === null) {
                reject(new Error("Socket is not connected"));
                return;
            }

            const id = Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);

            this.once(event + "@ack:" + id, data => {
                resolve(data.result);
            });
            this._socket.send(JSON.stringify({event, data, ackId: id}));
        });
    }

    /**
     * Adds an event listener to the client.
     * 
     * @param event The event to listen to.
     * @param handler The handler to call when the event is received.
     */
    public on(event: string, handler: EventHandler) {
        if (this._handlers[event] === undefined) {
            this._handlers[event] = [];
        }
        this._handlers[event].push(handler);
    }

    /**
     * Removes an event handler from the client.
     * 
     * @param event The event to remove the handler from.
     * @param handler The handler to remove.
     */
    public off(event: string, handler: EventHandler) {
        if (this._handlers[event] === undefined) {
            return;
        }

        const index = this._handlers[event].indexOf(handler);
        if (index !== -1) {
            this._handlers[event].splice(index, 1);
        }
    }

    /**
     * Adds a one-time event listener to the client.
     * 
     * @param event The event to listen to.
     * @param handler The handler to call when the event is received.
     */
    public once(event: string, handler: EventHandler) {
        const onceHandler = (data: EventData) => {
            handler(data);
            this.off(event, onceHandler);
        };
        this.on(event, onceHandler);
    }

    /**
     * Gets called when the websocket connection was established but the handshake not yet completed.
     * 
     * @param handler The handler to call when the event is received.
     */
    public onPreConnected(handler: () => void) {
        this._preConnectedHandler = handler;
    }

    /**
     * Gets called when the client is fully connected and the handshake completed.
     * 
     * @param handler The handler to call when the event is received.
     */
    public onConnected(handler: () => void) {
        this._connectedHandler = handler;
    }

    /**
     * Gets called when the client is disconnected from the server.
     * 
     * @param handler The handler to call when the event is received.
     */
    public onDisconnected(handler: () => void) {
        this._disconnectedHandler = handler;
    }

    /**
     * Returns the server given client id or an empty string if the handshake was not yet completed.
     */
    public get id(): string {
        return this._id;
    }

    private connect() {
        this._id = "";
        this._socket = new WebSocket(this._url);
        this._socket.onopen = () => this.onOpen();
        this._socket.onclose = () => this.onClose();
        this._socket.onmessage = (message) => this.onMessage(message);
    }

    private onOpen() {
        if (this._preConnectedHandler !== null) {
            this._preConnectedHandler();
        }
    }

    private onClose() {
        if (this._disconnectedHandler !== null) {
            this._disconnectedHandler();
        }
        setTimeout(() => this.connect(), this._reconnectTimeout);
    }

    private onMessage(message: MessageEvent) {
        const event = JSON.parse(message.data);
        const eventName = event.event;
        const eventData: EventData = event.data;

        if (eventName === "#handshake") {
            if (this._id !== "") {
                console.error("Received handshake event but already connected");
                return;
            }

            this._id = eventData.clientId as string;
            if (this._connectedHandler !== null) {
                this._connectedHandler();
            }
            return;
        }

        if (this._handlers[eventName] === undefined) {
            return;
        }

        for (const handler of this._handlers[eventName]) {
            handler(eventData);
        }
    }
}