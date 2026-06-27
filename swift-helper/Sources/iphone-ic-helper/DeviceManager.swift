import Foundation
import ImageCaptureCore
import UniformTypeIdentifiers

enum DeviceManagerError: Error, CustomStringConvertible {
    case noMatchingCameraDevice
    case sessionOpenFailed(Error)

    var description: String {
        switch self {
        case .noMatchingCameraDevice:
            return "no matching camera device found"
        case .sessionOpenFailed(let error):
            return "session open failed: \(error.localizedDescription)"
        }
    }

    var isUnlockError: Bool {
        switch self {
        case .sessionOpenFailed(let error as NSError):
            return error.domain == "com.apple.ImageCaptureCore" && error.code == -9943
        default:
            return false
        }
    }
}

class DeviceManager: NSObject, ICDeviceBrowserDelegate, ICDeviceDelegate {

    private let browser = ICDeviceBrowser()
    private var devices: [ICDevice] = []
    private var listCompletion: ((ListResponse) -> Void)?
    private var downloadCompletion: ((Bool) -> Void)?
    private var currentDevice: ICDevice?

    override init() {
        super.init()
        browser.delegate = self
    }

    func identify(completion: @escaping (IdentifyResponse) -> Void) {
        browser.start()
        DispatchQueue.main.asyncAfter(deadline: .now() + 5.0) {
            let response = self.buildIdentifyResponse()
            self.browser.stop()
            completion(response)
        }
    }

    private func buildIdentifyResponse() -> IdentifyResponse {
        let deviceDTOs = self.devices.map { device in
            Device(
                uuid: device.uuidString ?? "unknown",
                productName: device.productKind ?? "iPhone",
                deviceName: device.name ?? "iPhone",
                isTrusted: true
            )
        }
        return IdentifyResponse(devices: deviceDTOs)
    }

    func list(deviceUUID: String, completion: @escaping (Result<ListResponse, DeviceManagerError>) -> Void) {
        browser.start()
        findDeviceWithTimeout(deviceUUID: deviceUUID, timeout: 5.0) { device in
            guard let device = device, let camera = device as? ICCameraDevice else {
                FileHandle.standardError.write(Data("list: no matching camera device found\n".utf8))
                self.browser.stop()
                completion(.failure(.noMatchingCameraDevice))
                return
            }

            self.currentDevice = camera
            FileHandle.standardError.write(Data("list: device found, hasOpenSession=\(camera.hasOpenSession)\n".utf8))

            if camera.hasOpenSession {
                self.waitForContent(camera: camera) { response in
                    self.browser.stop()
                    completion(.success(response))
                }
            } else {
                self.requestOpenSessionWithRetry(camera: camera, attemptsRemaining: 5) { error in
                    if let error = error {
                        FileHandle.standardError.write(Data("list: session error: \(error)\n".utf8))
                        self.browser.stop()
                        completion(.failure(.sessionOpenFailed(error)))
                        return
                    }
                    FileHandle.standardError.write(Data("list: session opened, waiting for content\n".utf8))
                    self.waitForContent(camera: camera) { response in
                        self.browser.stop()
                        completion(.success(response))
                    }
                }
            }
        }
    }

    private func requestOpenSessionWithRetry(camera: ICCameraDevice, attemptsRemaining: Int, completion: @escaping (Error?) -> Void) {
        camera.requestOpenSession { error in
            guard let error = error else {
                completion(nil)
                return
            }

            let managerError = DeviceManagerError.sessionOpenFailed(error)
            if attemptsRemaining > 1 && managerError.isUnlockError {
                FileHandle.standardError.write(Data("session unlock error; retrying in 1s (\(attemptsRemaining - 1) attempts left)\n".utf8))
                DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                    self.requestOpenSessionWithRetry(camera: camera, attemptsRemaining: attemptsRemaining - 1, completion: completion)
                }
                return
            }

            completion(error)
        }
    }

    private func waitForContent(camera: ICCameraDevice, completion: @escaping (ListResponse) -> Void) {
        waitForContent(camera: camera, startedAt: Date(), timeout: 60.0, completion: completion)
    }

    private func waitForContent(camera: ICCameraDevice, startedAt: Date, timeout: TimeInterval, completion: @escaping (ListResponse) -> Void) {
        let pct = camera.contentCatalogPercentCompleted
        let elapsed = Date().timeIntervalSince(startedAt)
        FileHandle.standardError.write(Data("content catalog: \(pct)% after \(String(format: "%.1f", elapsed))s\n".utf8))

        if pct == 100 || elapsed >= timeout {
            let response = self.deliverList(camera: camera)
            FileHandle.standardError.write(Data("delivering \(response.files.count) files\n".utf8))
            completion(response)
            return
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
            self.waitForContent(camera: camera, startedAt: startedAt, timeout: timeout, completion: completion)
        }
    }

    private func deliverList(camera: ICCameraDevice) -> ListResponse {
        let files = camera.mediaFiles ?? []
        let entries = files.compactMap { item -> FileEntry? in
            guard let file = item as? ICCameraFile else { return nil }
            return FileEntry(
                handle: file.name ?? "",
                name: file.name ?? "",
                size: Int64(file.fileSize),
                created: self.formatDate(file.creationDate),
                mimeType: self.mimeType(for: file),
                livePhotoPair: self.findLivePhotoPair(for: file, in: files)
            )
        }
        return ListResponse(files: entries)
    }

    func download(deviceUUID: String, handle: String, toPath: String, completion: @escaping (Bool) -> Void) {
        browser.start()
        findDeviceWithTimeout(deviceUUID: deviceUUID, timeout: 5.0) { device in
            guard let camera = device as? ICCameraDevice else {
                self.browser.stop()
                completion(false)
                return
            }

            if !camera.hasOpenSession {
                self.requestOpenSessionWithRetry(camera: camera, attemptsRemaining: 5) { error in
                    if let error = error {
                        FileHandle.standardError.write(Data("download: session error: \(error)\n".utf8))
                        self.browser.stop()
                        completion(false)
                        return
                    }
                    FileHandle.standardError.write(Data("download: session opened\n".utf8))
                    self.performDownload(camera: camera, handle: handle, toPath: toPath, completion: completion)
                }
            } else {
                self.performDownload(camera: camera, handle: handle, toPath: toPath, completion: completion)
            }
        }
    }

    private func performDownload(camera: ICCameraDevice, handle: String, toPath: String, completion: @escaping (Bool) -> Void) {
        let files = camera.mediaFiles ?? []
        guard let file = files.compactMap({ $0 as? ICCameraFile }).first(where: { $0.name == handle }) else {
            self.browser.stop()
            completion(false)
            return
        }

        let url = URL(fileURLWithPath: toPath)
        let downloadsDir = url.deletingLastPathComponent()
        let options: [ICDownloadOption: Any] = [
            .downloadsDirectoryURL: downloadsDir,
            .saveAsFilename: url.lastPathComponent
        ]

        file.requestDownload(options: options) { _, error in
            self.browser.stop()
            if let error = error {
                FileHandle.standardError.write(Data("download error: \(error)\n".utf8))
                completion(false)
            } else {
                completion(true)
            }
        }
    }

    private func findDeviceWithTimeout(deviceUUID: String, timeout: TimeInterval, completion: @escaping (ICDevice?) -> Void) {
        if let device = findDevice(byUUID: deviceUUID) {
            completion(device)
            return
        }

        let startTime = Date()
        let checkInterval: TimeInterval = 0.5

        func check() {
            if let device = findDevice(byUUID: deviceUUID) {
                completion(device)
            } else if Date().timeIntervalSince(startTime) >= timeout {
                completion(nil)
            } else {
                DispatchQueue.main.asyncAfter(deadline: .now() + checkInterval) {
                    check()
                }
            }
        }

        check()
    }

    private func findDevice(byUUID uuid: String) -> ICDevice? {
        return devices.first { $0.uuidString == uuid }
    }

    private func formatDate(_ date: Date?) -> String {
        guard let date = date else { return "" }
        let formatter = ISO8601DateFormatter()
        return formatter.string(from: date)
    }

    private func mimeType(for file: ICCameraFile) -> String {
        let uti = file.uti ?? ""
        if #available(macOS 11.0, *), let type = UTType(uti) {
            return type.preferredMIMEType ?? ""
        }
        switch uti {
        case "public.jpeg": return "image/jpeg"
        case "public.heic": return "image/heic"
        case "public.mpeg-4": return "video/mp4"
        case "com.apple.quicktime-movie": return "video/quicktime"
        default: return ""
        }
    }

    private func findLivePhotoPair(for file: ICCameraFile, in allFiles: [ICCameraItem]) -> String? {
        guard file.uti == "public.heic" else { return nil }
        let baseName = (file.name ?? "").replacingOccurrences(of: ".HEIC", with: "")
        let movName = baseName + ".MOV"
        return allFiles.contains { $0.name == movName } ? movName : nil
    }

    // MARK: - ICDeviceBrowserDelegate

    func deviceBrowser(_ browser: ICDeviceBrowser, didAdd device: ICDevice, moreComing: Bool) {
        devices.append(device)
        device.delegate = self
    }

    func deviceBrowser(_ browser: ICDeviceBrowser, didRemove device: ICDevice, moreGoing: Bool) {
        devices.removeAll { $0 === device }
    }

    // MARK: - ICDeviceDelegate

    func didRemove(_ device: ICDevice) {
        devices.removeAll { $0 === device }
    }

    func device(_ device: ICDevice, didOpenSessionWithError error: Error?) {
        if let error = error {
            FileHandle.standardError.write(Data("session error: \(error)\n".utf8))
        }
    }

    func deviceDidBecomeReady(_ device: ICDevice) {
    }

    func device(_ device: ICDevice, didCloseSessionWithError error: Error?) {
    }
}
