
export class WebSocketManager {
    constructor(onMessageCallback, onOpenCallback, onCloseCallback) {
        this.ws = null;
        this.onMessageCallback = onMessageCallback;
        this.onOpenCallback = onOpenCallback;
        this.onCloseCallback = onCloseCallback;
    }

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            if (this.onOpenCallback) {
                this.onOpenCallback();
            }
        };
        
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (this.onMessageCallback) {
                this.onMessageCallback(data);
            }
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            if (this.onCloseCallback) {
                this.onCloseCallback();
            }
            
            // Reconnect after 3 seconds
            setTimeout(() => this.connect(), 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }
}
