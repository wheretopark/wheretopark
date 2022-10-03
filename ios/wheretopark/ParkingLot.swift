//
//  ParkingLot.swift
//  iosApp
//
//  Created by Grzegorz Barański on 23/08/2022.
//  Copyright © 2022 orgName. All rights reserved.
//

import Foundation
import shared
import CoreLocation
import PhoneNumberKit

extension Coordinate {
    var coordinate: CLLocationCoordinate2D {
        CLLocationCoordinate2D(latitude: latitude, longitude: longitude)
    }
    
    func distance(from: CLLocation) -> CLLocationDistance {
        let pointLocation = CLLocation(
            latitude: self.latitude,
            longitude: self.longitude
        )
        return from.distance(from: pointLocation)
    }
}

extension ParkingLotResource {
    var components: URLComponents {
        URLComponents(string: self.url)!
    }
}


extension ParkingLotPricingRule {
    var durationString: String {
        let components = self.durationComponents()
        let duration = DateComponents(
            day: Int(components.days),
            hour: Int(components.hours),
            minute: Int(components.minutes),
            second: Int(components.seconds),
            nanosecond: Int(components.nanoseconds)
        )
        let durationFormatter = DateComponentsFormatter()
        durationFormatter.unitsStyle = .full
        return durationFormatter.string(from: duration)!
    }
}


extension ParkingLotMetadata {
    var commentForLocale: String? {
        let languageCode = Locale.current.language.languageCode?.identifier ?? "en"
        let comment = comment[languageCode] ?? comment ["en"]
        return comment
    }
}
