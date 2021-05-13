Scripts in this folder are building an image that is used by Stork
in GitLab CI to run system tests in libvirt VM.

GitLab does not provide a runner that can handle libvirt so we are
using a custom runner. This custom runner is pulling the image from
GitLab Docker registry and then use it in libvirt to set up a VM for
testing. So the image pulled from registry is Docker image that
stores inside a qcow2 image that can be used by libvirt.

So this scripts first build qcow2 image, wrap it in Docker image and then
upload it to GitLab Docker registry.

To build a new image and upload it to GitLab registry invoke:

  $ ./build.sh
