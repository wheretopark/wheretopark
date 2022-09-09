package app.wheretopark.storekeeper

import app.wheretopark.shared.*
import com.auth0.jwt.JWT
import com.auth0.jwt.algorithms.Algorithm
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.json
import io.ktor.server.application.*
import io.ktor.server.auth.*
import io.ktor.server.auth.jwt.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.plugins.autohead.*
import io.ktor.server.plugins.callloging.*
import io.ktor.server.plugins.contentnegotiation.ContentNegotiation
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import java.net.URI

data class Config(
    val jwtSecret: String,
    val storeURI: URI,
    val port: Int,
)

fun loadConfig() = Config(
    jwtSecret = System.getenv("JWT_SECRET")!!,
    storeURI = URI(System.getenv("STORE_URI") ?: "memory:/"),
    port = System.getenv("PORT")?.toInt() ?: 8080,
)


fun main() {
    val config = loadConfig()
    val store = when (config.storeURI.scheme) {
        "memory" -> {
            MemoryStore()
        }

        "redis" -> {
            RedisStore(config.storeURI.host, if (config.storeURI.port == -1) 6379 else config.storeURI.port)
        }

        else -> {
            throw IllegalArgumentException("Unknown store scheme: ${config.storeURI.scheme}")
        }
    }
    embeddedServer(Netty, port = config.port) {
        configure(store, config)
    }.start(wait = true)
}

// TODO: Find a cleaner way to do it
suspend fun validateScope(call: ApplicationCall, accessType: AccessType): Boolean {
    val principal = call.principal<JWTPrincipal>()
    val scope = principal?.getClaim("scope", String::class) ?: ""
    val accessScope = decodeAccessScope(scope)
    return if (accessScope.contains(accessType)) {
        true
    } else {
        call.respond(HttpStatusCode.Unauthorized, "missing access ${accessType.name} in scope $scope")
        false
    }

}

fun Application.configure(store: Store, config: Config) {
    install(ContentNegotiation) {
        json()
    }
    install(CallLogging)
    install(AutoHeadResponse)
    install(Authentication) {
        jwt("auth-jwt") {
            realm = "Storekeeper service"
            verifier(
                JWT.require(Algorithm.HMAC512(config.jwtSecret)).build()
            )
            validate { credential ->
                JWTPrincipal(credential.payload)
            }
        }
    }
    routing {
        get("/health-check") {
            call.respond("Hello, World!")
        }
        authenticate("auth-jwt") {
            route("/parking-lot") {
                route("/metadata") {
                    get {
                        if (!validateScope(call, AccessType.ReadMetadata)) return@get
                        call.respond(store.getMetadatas())
                    }
                    post {
                        if (!validateScope(call, AccessType.WriteMetadata)) return@post
                        val newStates = call.receive<Map<ParkingLotID, ParkingLotState>>()
                        store.updateStates(newStates)
                        call.respondText("updated ${newStates.count()} states", status = HttpStatusCode.Accepted)
                    }
                }
                route("/state") {
                    get {
                        if (!validateScope(call, AccessType.ReadState)) return@get
                        call.respond(store.getStates())
                    }
                    post {
                        if (!validateScope(call, AccessType.WriteState)) return@post
                        val newMetadatas = call.receive<Map<ParkingLotID, ParkingLotMetadata>>()
                        store.updateMetadatas(newMetadatas)
                        call.respondText("updated ${newMetadatas.count()} metadatas", status = HttpStatusCode.Accepted)
                    }
                }
            }
        }
    }
}