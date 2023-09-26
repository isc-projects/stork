
#############
### Tasks ###
#############

namespace :push do
    # Build the image from the :source file.
    # The image name is defined by the :target argument.
    # The tag is defined by the TAG environment variable. The allowed values
    # are positive integers or the 'latest` keyword.
    # The image is pushed to the registry only if the PUSH environment variable
    # has "true" value.
    # The cache may be disabled by the CACHE environment variable set to "false".
    task :build_and_push, [:source, :target, :with_arm] => [DOCKER, DOCKER_BUILDX] do |t, args|
        build_opts = []
        build_platforms = [
            "--platform", "linux/amd64",
        ]
        if args[:with_arm]
            build_platforms.append "--platform", "linux/arm64/v8"
        end

        # Cache options.
        if ENV["CACHE"] == "false"
            build_opts.append "--no-cache"
        end

        # Prepare the target.
        tag = ENV["TAG"]
        if tag.nil?
            fail "You must specify the TAG environment variable"
        end
        tag = tag.rstrip
    
        if tag.to_i.to_s != tag && tag != "latest"
            fail "Wrong tag: #{tag}. Only integer numbers are allowed or 'latest'."
        end
    
        target = "#{args[:target]}:#{tag}"

        # Determine operation to perform.
        # --push or --load
        post_build_opts = []
        # All build platform or none (current machine platform)
        post_build_platforms = []

        push = ENV["PUSH"]
        if push.nil?
            fail "You must specify the operation: PUSH=true (for push) or PUSH=false (for build only)"
        end
        if push == "true"
            sh DOCKER, "login", "registry.gitlab.isc.org"
            post_build_opts.append "--push"
            # Load doesn't support multi-platform manifest.
            post_build_platforms = build_platforms
        else
            post_build_opts.append "--load"
        end

        # Execture commands.
        # We build the CI images using the buildx plugin instead of the standard
        # build command to enable multi-architecture build on the machines
        # that aren't multi-architectural (standard computers).

        # Constant builder name to re-use build cache.
        builder_name = "stork"
        _, status = Open3.capture2 *DOCKER_BUILDX, "use", builder_name
        if status != 0
            sh *DOCKER_BUILDX, "create", "--use", "--name", builder_name
        end

        opts = [
            "-f", args[:source],
            "-t", target,
            "docker/"
        ]

        # Build for all platforms
        sh *DOCKER_BUILDX, "build",
            *build_opts,
            *build_platforms,
            *opts

        # Load or push
        sh *DOCKER_BUILDX, "build",
            *post_build_opts,
            *post_build_platforms,
            *opts
    end

    desc 'Prepare CI-purpose image based on Debian.
        TAG - number used as the image tag or "latest" keyword - required
        CACHE - allow using cached image layers - default: true
        PUSH - push image to the registry - required'
    task :base_deb do
        Rake::Task["push:build_and_push"].invoke(
            "docker/images/ci/debian.Dockerfile",
            "registry.gitlab.isc.org/isc-projects/stork/ci-base",
            true
        )
    end

    desc 'Prepare CI-purpose image based on RHEL.
        TAG - number used as the image tag or "latest" keyword - required
        CACHE - allow using cached image layers - default: true
        PUSH - push image to the registry - required'
    task :base_rhel do
        Rake::Task["push:build_and_push"].invoke(
            "docker/images/ci/redhat-ubi8.Dockerfile",
            "registry.gitlab.isc.org/isc-projects/stork/pkgs-redhat-ubi8",
            true
        )
    end

    desc 'Prepare CI-purpose image based on Alpine.
        TAG - number used as the image tag or "latest" keyword - required
        CACHE - allow using cached image layers - default: true
        PUSH - push image to the registry - required'
    task :base_alpine do
        Rake::Task["push:build_and_push"].invoke(
            "docker/images/ci/alpine.Dockerfile",
            "registry.gitlab.isc.org/isc-projects/stork/pkgs-alpine",
            true
        )
    end

    desc 'Prepare CI-purpose image based on the official Docker image.
        TAG - number used as the image tag or "latest" keyword - required
        CACHE - allow using cached image layers - default: true
        PUSH - push image to the registry - required'
    task :base_compose do
        Rake::Task["push:build_and_push"].invoke(
            "docker/images/ci/compose.Dockerfile",
            "registry.gitlab.isc.org/isc-projects/stork/pkgs-compose",
            false
        )
    end

    desc 'Prepare CI-purpose image with the CloudSmith tools.
        TAG - number used as the image tag or "latest" keyword - required
        CACHE - allow using cached image layers - default: true
        PUSH - push image to the registry - required'
    task :base_cloudsmith do
        Rake::Task["push:build_and_push"].invoke(
            "docker/images/ci/cloudsmith.Dockerfile",
            "registry.gitlab.isc.org/isc-projects/stork/pkgs-cloudsmith",
            false
        )
    end
end

namespace :check do
    desc 'Check the external dependencies related to the distribution'
    task :registry do
        check_deps(__FILE__)
    end
end
