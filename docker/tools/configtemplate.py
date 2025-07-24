import os
import string
import platform
import sys

DEFAULT_INPUT_DIR = "/tmp/keacfg/"
DEFAULT_OUTPUT_DIR = "/etc/kea/"


def get_platform_triple():
    """Get the 'platform triple' for the machine running this script.

    The platform triple is a three part string identifying the CPU, Kernel, and
    ABI of the environment.  Common examples include:
      - x86_64-linux-gnu
      - x86_64-linux-musl
      - aarch64-darwin-apple
      - riscv-linux-musl
    """
    cpu = platform.machine()
    kernel = sys.platform.lower()
    abi = "gnu"  # <- Improve this when the demo needs to run with musl, etc.
    return f"{cpu}-{kernel}-{abi}"


def parse_args(args: [str]) -> (str, str, str):
    """Parse the expected arguments to this script.
    :param args: The list of command-line arguments to this tool.
    :return: A tuple with the filename to template, the input directory in which
    to look for the file, and the output directory in which to write the
    templated file.
    """
    if len(args) == 2:
        return args[1], DEFAULT_INPUT_DIR, DEFAULT_OUTPUT_DIR
    if len(args) == 4:
        return args[1], args[2], args[3]
    raise ValueError(
        "usage: configtemplate.py (kea-dhcp4.conf|kea-dhcp6.conf) [INPUT_DIR OUTPUT_DIR]"
    )


def main(args: [str]):
    """Run the configtemplate program.
    :param args: Command-line arguments.  See parse_args().
    """
    target_file, input_dir, output_dir = parse_args(args)
    os.makedirs(output_dir, exist_ok=True)
    # Using os.listdir because file type information is not necessary.
    for entry in os.listdir(input_dir):
        triple = get_platform_triple()
        if entry == target_file:
            tmpl = None
            infile_name = os.path.join(input_dir, entry)
            with open(infile_name, "r", encoding="utf-8") as infile:
                tmpl = string.Template(infile.read())
            result = tmpl.safe_substitute(KEA_PLATFORM_TRIPLE=triple)
            outfile_name = os.path.join(output_dir, entry)
            with open(outfile_name, "w", encoding="utf-8") as outfile:
                outfile.write(result)
            print(f"configtemplate.py: templated '{infile_name}' to '{outfile_name}'")


if __name__ == "__main__":
    main(sys.argv)
