import Foundation

enum Commands {

    static func identify() {
        let manager = DeviceManager()
        manager.identify { response in
            outputJSON(response)
            exit(0)
        }
    }

    static func list(args: [String]) {
        var deviceUUID: String?
        var i = 0
        while i < args.count {
            if args[i] == "--device" && i + 1 < args.count {
                deviceUUID = args[i + 1]
                i += 2
            } else {
                i += 1
            }
        }

        guard let uuid = deviceUUID else {
            outputError("missing --device argument")
            exit(1)
        }

        let manager = DeviceManager()
        manager.list(deviceUUID: uuid) { result in
            switch result {
            case .success(let response):
                outputJSON(response)
                exit(0)
            case .failure(let error):
                outputError(error.description)
                exit(1)
            }
        }
    }

    static func download(args: [String]) {
        var deviceUUID: String?
        var handle: String?
        var toPath: String?
        var i = 0
        while i < args.count {
            switch args[i] {
            case "--device" where i + 1 < args.count:
                deviceUUID = args[i + 1]; i += 2
            case "--handle" where i + 1 < args.count:
                handle = args[i + 1]; i += 2
            case "--to" where i + 1 < args.count:
                toPath = args[i + 1]; i += 2
            default:
                i += 1
            }
        }

        guard let uuid = deviceUUID, let h = handle, let to = toPath else {
            outputError("missing required arguments: --device <uuid> --handle <h> --to <path>")
            exit(1)
        }

        let manager = DeviceManager()
        manager.download(deviceUUID: uuid, handle: h, toPath: to) { success in
            if success {
                exit(0)
            } else {
                exit(1)
            }
        }
    }
}
