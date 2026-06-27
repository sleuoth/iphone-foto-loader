import Foundation

struct Device: Codable {
    let uuid: String
    let productName: String
    let deviceName: String
    let isTrusted: Bool
}

struct IdentifyResponse: Codable {
    let devices: [Device]
}

struct FileEntry: Codable {
    let handle: String
    let name: String
    let size: Int64
    let created: String
    let mimeType: String
    let livePhotoPair: String?

    enum CodingKeys: String, CodingKey {
        case handle, name, size, created, mimeType, livePhotoPair
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(handle, forKey: .handle)
        try container.encode(name, forKey: .name)
        try container.encode(size, forKey: .size)
        try container.encode(created, forKey: .created)
        try container.encode(mimeType, forKey: .mimeType)
        try container.encode(livePhotoPair, forKey: .livePhotoPair)
    }
}

struct ListResponse: Codable {
    let files: [FileEntry]
}

struct ErrorResponse: Codable {
    let error: String
}

func encodeJSON<T: Encodable>(_ value: T) -> String {
    let encoder = JSONEncoder()
    encoder.outputFormatting = []
    if let data = try? encoder.encode(value) {
        return String(data: data, encoding: .utf8) ?? "{}"
    }
    return "{}"
}

func outputJSON<T: Encodable>(_ value: T) {
    print(encodeJSON(value))
}

func outputError(_ message: String) {
    print(encodeJSON(ErrorResponse(error: message)))
    FileHandle.standardError.write(Data("ERROR: \(message)\n".utf8))
}
