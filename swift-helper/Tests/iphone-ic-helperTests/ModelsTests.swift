import XCTest
@testable import iphone_ic_helper

final class ModelsTests: XCTestCase {

    func testIdentifyResponseEncode() throws {
        let device = Device(uuid: "test-uuid", productName: "iPhone 15 Pro", deviceName: "Test", isTrusted: true)
        let resp = IdentifyResponse(devices: [device])
        let json = encodeJSON(resp)

        XCTAssertTrue(json.contains("\"uuid\":\"test-uuid\""))
        XCTAssertTrue(json.contains("\"productName\":\"iPhone 15 Pro\""))
        XCTAssertTrue(json.contains("\"isTrusted\":true"))
    }

    func testListResponseEncodeWithLivePhotoPair() throws {
        let file = FileEntry(
            handle: "h1", name: "IMG_1234.HEIC", size: 1234,
            created: "2026-06-27T10:00:00Z", mimeType: "image/heic",
            livePhotoPair: "IMG_1234.MOV"
        )
        let resp = ListResponse(files: [file])
        let json = encodeJSON(resp)

        XCTAssertTrue(json.contains("\"livePhotoPair\":\"IMG_1234.MOV\""))
        XCTAssertTrue(json.contains("\"handle\":\"h1\""))
    }

    func testListResponseEncodeNullLivePhotoPair() throws {
        let file = FileEntry(
            handle: "h2", name: "IMG_1235.MOV", size: 5678,
            created: "2026-06-27T10:01:00Z", mimeType: "video/quicktime",
            livePhotoPair: nil
        )
        let resp = ListResponse(files: [file])
        let json = encodeJSON(resp)

        XCTAssertTrue(json.contains("\"livePhotoPair\":null"))
    }

    func testIdentifyResponseDecode() throws {
        let json = """
        {"devices":[{"uuid":"abc","productName":"iPhone 14","deviceName":"My Phone","isTrusted":false}]}
        """.data(using: .utf8)!

        let resp = try JSONDecoder().decode(IdentifyResponse.self, from: json)
        XCTAssertEqual(resp.devices.count, 1)
        XCTAssertEqual(resp.devices[0].uuid, "abc")
        XCTAssertEqual(resp.devices[0].productName, "iPhone 14")
        XCTAssertFalse(resp.devices[0].isTrusted)
    }

    func testListResponseDecode() throws {
        let json = """
        {"files":[{"handle":"h1","name":"test.heic","size":100,"created":"2026-01-01T00:00:00Z","mimeType":"image/heic","livePhotoPair":null}]}
        """.data(using: .utf8)!

        let resp = try JSONDecoder().decode(ListResponse.self, from: json)
        XCTAssertEqual(resp.files.count, 1)
        XCTAssertEqual(resp.files[0].handle, "h1")
        XCTAssertNil(resp.files[0].livePhotoPair)
    }
}
