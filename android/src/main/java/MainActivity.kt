package app.wheretopark.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.annotation.RequiresApi
import androidx.compose.runtime.*
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.wheretopark.android.ui.theme.WheretoparkTheme
import app.wheretopark.shared.*
import kotlinx.coroutines.launch


class MainActivity : ComponentActivity() {
    @RequiresApi(33)
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val clientID = getString(R.string.client_id)
        val clientSecret = getString(R.string.client_secret)
        val parkingLotViewModel = ParkingLotViewModel(clientID, clientSecret)
        setContent {
            MainView(parkingLotViewModel)
        }
    }
}

@RequiresApi(33)
@Composable
fun MainView(parkingLotViewModel: ParkingLotViewModel) {
    WheretoparkTheme {
        LaunchedEffect(Unit, block = {
            parkingLotViewModel.fetchParkingLots()
        })

        DetailsBottomSheet(parkingLotViewModel) {
            ListBottomSheet(parkingLotViewModel) {
                MapView(parkingLotViewModel)
            }
        }
    }

}

class ParkingLotViewModel(clientID: String, clientSecret: String) : ViewModel() {
    val parkingLots = mutableStateMapOf<ParkingLotID, ParkingLot>()
    var selectedParkingLotID by mutableStateOf<ParkingLotID?>(null)
    private val authorizationClient = AuthorizationClient(clientID = clientID, clientSecret = clientSecret)
    private val storekeeperClient = StorekeeperClient(
        authorizationClient = authorizationClient,
        accessScope = setOf(
            AccessType.ReadMetadata, AccessType.ReadState, AccessType.ReadStatus
        )
    )

    fun fetchParkingLots() {
        println("fetching parking lots")
        viewModelScope.launch {
            val newParkingLots = storekeeperClient.parkingLots()
            println("retrieved ${newParkingLots.count()} parking lots")
            parkingLots.clear()
            newParkingLots.forEach { (key, value) ->
                parkingLots[key] = value
            }
        }
    }
}

//@Preview(showBackground = true)
//@Composable
//fun DefaultPreview() {
//    WheretoparkTheme {
//        Greeting("Android")
//    }
//}