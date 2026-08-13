package dev.gad.intellij.debug.dap

import com.google.gson.Gson
import com.google.gson.JsonObject
import com.intellij.openapi.diagnostic.thisLogger
import java.io.BufferedInputStream
import java.io.InputStream
import java.io.OutputStream
import java.nio.charset.StandardCharsets
import java.util.concurrent.CompletableFuture
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicInteger

/**
 * A minimal Debug Adapter Protocol client speaking `Content-Length`-framed JSON
 * over the adapter process's stdio. Requests return a future resolved with the
 * response `body`; events are pushed to [listener].
 */
class DapClient(
    private val input: InputStream,
    private val output: OutputStream,
    private val listener: Listener,
) {
    interface Listener {
        /** A DAP event, e.g. `stopped`, `output`, `terminated`. */
        fun onEvent(event: String, body: JsonObject?)

        /** The adapter stream closed. */
        fun onClosed()
    }

    private val gson = Gson()
    private val seq = AtomicInteger(0)
    private val pending = ConcurrentHashMap<Int, CompletableFuture<JsonObject?>>()
    @Volatile private var closed = false
    private var reader: Thread? = null

    fun start() {
        reader = Thread({ readLoop() }, "gad-dap-reader").apply { isDaemon = true; start() }
    }

    /** Send a request and complete the future with its response body (or null). */
    fun request(command: String, arguments: JsonObject? = null): CompletableFuture<JsonObject?> {
        val s = seq.incrementAndGet()
        val msg = JsonObject().apply {
            addProperty("seq", s)
            addProperty("type", "request")
            addProperty("command", command)
            if (arguments != null) add("arguments", arguments)
        }
        val future = CompletableFuture<JsonObject?>()
        pending[s] = future
        write(msg)
        return future
    }

    fun close() {
        if (closed) return
        closed = true
        try { output.close() } catch (_: Exception) {}
        try { input.close() } catch (_: Exception) {}
        pending.values.forEach { it.cancel(true) }
        pending.clear()
    }

    @Synchronized
    private fun write(msg: JsonObject) {
        if (closed) return
        try {
            val body = gson.toJson(msg).toByteArray(StandardCharsets.UTF_8)
            output.write("Content-Length: ${body.size}\r\n\r\n".toByteArray(StandardCharsets.US_ASCII))
            output.write(body)
            output.flush()
        } catch (e: Exception) {
            thisLogger().debug("DAP write failed", e)
        }
    }

    private fun readLoop() {
        val stream = BufferedInputStream(input)
        try {
            while (!closed) {
                val length = readHeaderContentLength(stream) ?: break
                val buf = ByteArray(length)
                var read = 0
                while (read < length) {
                    val n = stream.read(buf, read, length - read)
                    if (n < 0) throw java.io.EOFException()
                    read += n
                }
                dispatch(gson.fromJson(String(buf, StandardCharsets.UTF_8), JsonObject::class.java))
            }
        } catch (_: Exception) {
            // stream closed / process ended
        } finally {
            if (!closed) listener.onClosed()
        }
    }

    private fun readHeaderContentLength(stream: InputStream): Int? {
        var length = -1
        val line = StringBuilder()
        while (true) {
            val c = stream.read()
            if (c < 0) return null
            if (c == '\n'.code) {
                val header = line.toString().trim()
                line.setLength(0)
                if (header.isEmpty()) break // end of headers
                val idx = header.indexOf(':')
                if (idx > 0 && header.substring(0, idx).trim().equals("Content-Length", true)) {
                    length = header.substring(idx + 1).trim().toIntOrNull() ?: -1
                }
            } else if (c != '\r'.code) {
                line.append(c.toChar())
            }
        }
        return if (length >= 0) length else null
    }

    private fun dispatch(msg: JsonObject) {
        when (msg.get("type")?.asString) {
            "response" -> {
                val reqSeq = msg.get("request_seq")?.asInt ?: return
                val future = pending.remove(reqSeq) ?: return
                val success = msg.get("success")?.asBoolean ?: false
                if (success) {
                    future.complete(msg.getAsJsonObject("body"))
                } else {
                    future.completeExceptionally(
                        DapError(msg.get("message")?.asString ?: "request failed"),
                    )
                }
            }
            "event" -> listener.onEvent(msg.get("event").asString, msg.getAsJsonObject("body"))
        }
    }
}

class DapError(message: String) : RuntimeException(message)
