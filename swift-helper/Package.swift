// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "iphone-ic-helper",
    platforms: [
        .macOS(.v10_15)
    ],
    targets: [
        .executableTarget(
            name: "iphone-ic-helper",
            path: "Sources/iphone-ic-helper"
        ),
        .testTarget(
            name: "iphone-ic-helperTests",
            dependencies: ["iphone-ic-helper"],
            path: "Tests/iphone-ic-helperTests"
        ),
    ]
)
