import XCTest
@testable import iphone_ic_helper

final class DeviceManagerErrorTests: XCTestCase {

    func testSessionOpenFailedDetectsUnlockErrorCode() {
        let nsError = NSError(
            domain: "com.apple.ImageCaptureCore",
            code: -9943,
            userInfo: [NSLocalizedDescriptionKey: "Please unlock \"Test iPhone\""]
        )

        let error = DeviceManagerError.sessionOpenFailed(nsError)

        XCTAssertTrue(error.isUnlockError)
        XCTAssertTrue(error.description.contains("Please unlock"))
    }
}
