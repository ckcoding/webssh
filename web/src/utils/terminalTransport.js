const ACK_BYTES = 64 * 1024
const INPUT_CHUNK_BYTES = 64 * 1024
const INPUT_WINDOW_BYTES = 256 * 1024
const INPUT_ACK_PREFIX = '\u0000inputack:'
const encoder = new TextEncoder()

// Acknowledge output only after xterm has parsed it. The server then applies
// backpressure to SSH output while keyboard input remains immediate.
export class TerminalTransport {
    constructor(socket) {
        this.socket = socket
        this.pendingAck = 0
        this.disposed = false
        this.disposables = []
        this.inputQueue = []
        this.inputQueueHead = 0
        this.pendingInput = 0
    }

    activate(terminal) {
        this.messageHandler = (event) => {
            if (typeof event.data === 'string' && event.data.startsWith(INPUT_ACK_PREFIX)) {
                const bytes = Number(event.data.slice(INPUT_ACK_PREFIX.length))
                if (Number.isInteger(bytes) && bytes > 0 && bytes <= this.pendingInput) {
                    this.pendingInput -= bytes
                    this.flushInput()
                }
                return
            }
            const data = typeof event.data === 'string' ? event.data : new Uint8Array(event.data)
            const bytes = typeof data === 'string' ? encoder.encode(data).byteLength : data.byteLength
            terminal.write(data, () => {
                if (this.disposed) {
                    return
                }
                this.pendingAck += bytes
                if (this.pendingAck >= ACK_BYTES && this.socket.readyState === 1) {
                    this.socket.send(`\u0000ack:${this.pendingAck}`)
                    this.pendingAck = 0
                }
            })
        }
        this.closeHandler = () => this.dispose()
        this.socket.addEventListener('message', this.messageHandler)
        this.socket.addEventListener('close', this.closeHandler)
        this.socket.addEventListener('error', this.closeHandler)
        this.disposables.push(terminal.onData(data => this.sendInput(encoder.encode(data))))
        this.disposables.push(terminal.onBinary(data => {
            const bytes = Uint8Array.from(data, char => char.charCodeAt(0) & 255)
            this.sendInput(bytes)
        }))
    }

    sendInput(bytes) {
        if (this.disposed || this.socket.readyState !== 1) {
            return
        }
        // Binary input is distinct from text control frames, including pasted
        // text that happens to start with "resize:" or an acknowledgement.
        this.inputQueue.push({ bytes, offset: 0 })
        this.flushInput()
    }

    flushInput() {
        while (this.inputQueueHead < this.inputQueue.length && this.pendingInput < INPUT_WINDOW_BYTES) {
            const item = this.inputQueue[this.inputQueueHead]
            const size = Math.min(INPUT_CHUNK_BYTES, item.bytes.byteLength - item.offset, INPUT_WINDOW_BYTES - this.pendingInput)
            this.socket.send(item.bytes.subarray(item.offset, item.offset + size))
            this.pendingInput += size
            item.offset += size
            if (item.offset === item.bytes.byteLength) {
                this.inputQueueHead += 1
            }
        }
        if (this.inputQueueHead === this.inputQueue.length) {
            this.inputQueue = []
            this.inputQueueHead = 0
        } else if (this.inputQueueHead > 64) {
            this.inputQueue.splice(0, this.inputQueueHead)
            this.inputQueueHead = 0
        }
    }

    dispose() {
        if (this.disposed) {
            return
        }
        this.disposed = true
        this.socket.removeEventListener('message', this.messageHandler)
        this.socket.removeEventListener('close', this.closeHandler)
        this.socket.removeEventListener('error', this.closeHandler)
        this.disposables.forEach(disposable => disposable.dispose())
        this.disposables = []
        this.pendingAck = 0
        this.pendingInput = 0
        this.inputQueue = []
        this.inputQueueHead = 0
    }
}
