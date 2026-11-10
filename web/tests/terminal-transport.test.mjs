import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('../src/utils/terminalTransport.js', import.meta.url), 'utf8')
const transportModule = import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`)

class Socket {
    readyState = 1
    sent = []
    listeners = new Map()
    addEventListener(type, fn) { this.listeners.set(type, fn) }
    removeEventListener(type) { this.listeners.delete(type) }
    send(data) { this.sent.push(data) }
    emit(type, data) { this.listeners.get(type)?.({ data }) }
}

function terminal() {
    return {
        output: [],
        callbacks: [],
        write(data, callback) { this.output.push(data); this.callbacks.push(callback) },
        onData(fn) { this.input = fn; return { dispose: () => { this.input = null } } },
        onBinary(fn) { this.binary = fn; return { dispose: () => { this.binary = null } } }
    }
}

test('parsed output releases flow control while keyboard input stays immediate', async () => {
    const { TerminalTransport } = await transportModule
    const socket = new Socket()
    const term = terminal()
    new TerminalTransport(socket).activate(term)
    socket.emit('message', new Uint8Array(32768).buffer)
    socket.emit('message', new Uint8Array(32768).buffer)
    assert.equal(socket.sent.length, 0)
    term.input('ls\r')
    assert.equal(Buffer.from(socket.sent[0]).toString(), 'ls\r')
    term.callbacks[0]()
    assert.equal(socket.sent.length, 1)
    term.callbacks[1]()
    assert.equal(socket.sent[1], '\u0000ack:65536')
})

test('UTF-8 split across output frames stays intact and counts bytes', async () => {
    const { TerminalTransport } = await transportModule
    const socket = new Socket()
    const term = terminal()
    new TerminalTransport(socket).activate(term)
    const data = Buffer.from('中文🙂'.repeat(10000))
    for (let offset = 0; offset < data.length; offset += 32767) {
        const chunk = Uint8Array.from(data.subarray(offset, offset + 32767))
        socket.emit('message', chunk.buffer)
    }
    assert.deepEqual(Buffer.concat(term.output.map(chunk => Buffer.from(chunk))), data)
    term.callbacks.forEach(callback => callback())
    assert.equal(socket.sent[0], '\u0000ack:98301')
})

test('large Unicode pastes are ordered binary chunks distinct from controls', async () => {
    const { TerminalTransport } = await transportModule
    const socket = new Socket()
    const term = terminal()
    new TerminalTransport(socket).activate(term)
    const text = 'resize:24:80\r' + '🙂中文'.repeat(200000)
    term.input(text)
    assert.equal(socket.sent.reduce((sum, frame) => sum + frame.byteLength, 0), 256 * 1024)
    let granted = 0
    while (socket.sent.reduce((sum, frame) => sum + frame.byteLength, 0) < Buffer.byteLength(text)) {
        const sent = socket.sent.reduce((sum, frame) => sum + frame.byteLength, 0)
        socket.emit('message', `\u0000inputack:${sent - granted}`)
        granted = sent
    }
    const sent = socket.sent.reduce((sum, frame) => sum + frame.byteLength, 0)
    socket.emit('message', `\u0000inputack:${sent - granted}`)
    assert(socket.sent.length > 1)
    assert(socket.sent.every(chunk => chunk instanceof Uint8Array && chunk.length <= 65536))
    assert.deepEqual(Buffer.concat(socket.sent.map(chunk => Buffer.from(chunk))), Buffer.from(text))
    term.binary('\u0080\u00ff')
    assert.deepEqual([...socket.sent.at(-1)], [128, 255])
    assert.equal(term.output.length, 0, 'input credits must not be printed in the terminal')
})

test('disconnect disposes listeners and prevents late acknowledgements', async () => {
    const { TerminalTransport } = await transportModule
    const socket = new Socket()
    const term = terminal()
    const transport = new TerminalTransport(socket)
    transport.activate(term)
    socket.emit('message', new Uint8Array(65536).buffer)
    socket.emit('close')
    transport.dispose()
    term.callbacks[0]()
    assert.equal(socket.sent.length, 0)
    assert.equal(socket.listeners.size, 0)
    assert.equal(term.input, null)
    assert.equal(term.binary, null)
})
