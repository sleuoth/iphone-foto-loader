import XCTest

final class DeviceManagerDownloadRetryTests: XCTestCase {

    func testDownloadUsesSessionOpenRetryHelper() throws {
        let sourceURL = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Sources/iphone-ic-helper/DeviceManager.swift")
        let source = try String(contentsOf: sourceURL)

        guard let downloadRange = source.range(of: "func download(deviceUUID: String, handle: String, toPath: String, completion: @escaping (Bool) -> Void)") else {
            return XCTFail("download function not found")
        }
        guard let performDownloadRange = source.range(of: "private func performDownload", range: downloadRange.upperBound..<source.endIndex) else {
            return XCTFail("performDownload function not found after download")
        }

        let downloadSource = source[downloadRange.lowerBound..<performDownloadRange.lowerBound]
        XCTAssertTrue(
            downloadSource.contains("requestOpenSessionWithRetry"),
            "download should use the same unlock/session retry handling as list"
        )
    }
}
