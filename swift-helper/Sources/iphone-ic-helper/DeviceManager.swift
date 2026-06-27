import Foundation
import ImageCaptureCore
import UniformTypeIdentifiers

class DeviceManager: NSObject, ICDeviceBrowserDelegate, ICDeviceDelegate {

    private let browser = ICDeviceBrowser()
    private var devices: [ICDevice] = []
    private var completionHandler: ((IdentifyResponse) -> Void)?
    private var listCompletion: ((ListResponse) -> Void)?
    private var downloadCompletion: ((Bool) -> Void)?
    private var currentDevice: ICDevice?

    override init() {
        super.init()
        browser.delegate = self
    }

    func identify(completion: @escaping (IdentifyResponse) -> Void) {
        self.completionHandler = completion
        browser.start()
        DispatchQueue.main.asyncAfter(deadline: .now() + 3.0) {
            self.browser.stop()
            let response = self.buildIdentifyResponse()
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

    func list(deviceUUID: String, completion: @escaping (ListResponse) -> Void) {
        self.listCompletion = completion

        browser.start()
        findDeviceWithTimeout(deviceUUID: deviceUUID, timeout: 3.0) { device in
            guard let device = device, let camera = device as? ICCameraDevice else {
                self.browser.stop()
                completion(ListResponse(files: []))
                return
            }

            self.currentDevice = camera

            if camera.hasOpenSession {
                self.waitForContent(camera: camera, completion: completion)
            } else {
                camera.requestOpenSession { error in
                    if let error = error {
                        FileHandle.standardError.write(Data("session error: \(error)\n".utf8))
                        self.browser.stop()
                        completion(ListResponse(files: []))
                        return
                    }
                    self.waitForContent(camera: camera, completion: completion)
                }
            }
        }
    }

    private func waitForContent(camera: ICCameraDevice, completion: @escaping (ListResponse) -> Void) {
        if camera.contentCatalogPercentCompleted == 100 {
            self.deliverList(camera: camera, completion: completion)
        } else {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                self.deliverList(camera: camera, completion: completion)
            }
        }
    }

    private func deliverList(camera: ICCameraDevice, completion: @escaping (ListResponse) -> Void) {
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
        completion(ListResponse(files: entries))
    }

    func download(deviceUUID: String, handle: String, toPath: String, completion: @escaping (Bool) -> Void) {
        self.downloadCompletion = completion

        browser.start()
        findDeviceWithTimeout(deviceUUID: deviceUUID, timeout: 3.0) { device in
            guard let camera = device as? ICCameraDevice else {
                self.browser.stop()
                completion(false)
                return
            }

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

    func device(_ device: ICDevice, didCloseSessionWithError error: Error?) {
        // Session closed
    }
}
