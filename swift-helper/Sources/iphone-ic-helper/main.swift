import Foundation

let args = CommandLine.arguments

guard args.count >= 2 else {
    outputError("no subcommand provided. usage: iphone-ic-helper <identify|list|download>")
    exit(1)
}

let subcommand = args[1]

switch subcommand {
case "identify":
    Commands.identify()
case "list":
    Commands.list(args: Array(args.dropFirst(2)))
case "download":
    Commands.download(args: Array(args.dropFirst(2)))
case "--help", "-h":
    print("usage: iphone-ic-helper <identify|list --device <uuid>|download --device <uuid> --handle <h> --to <path>>")
    exit(0)
default:
    outputError("unknown subcommand: \(subcommand)")
    exit(1)
}

RunLoop.main.run()
