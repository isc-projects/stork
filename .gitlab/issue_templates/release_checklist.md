# Stork Release Checklist

#### Legend

- `A.B.C`: the version being released

- `Stable Release`: the first release on a stable branch. `B` is an even number. `C` is 0.

- `Maintenance Release`: any release on a stable branch except the first. `B` is an even number. `C` is not 0.

- `Dev Release`: any release from the `master` and `main` branches. `B` is an odd number.

- `Security Release`: any release that contains changes that were assigned a CVE number.

#### General Guidelines

- <mark>Stable or Maintenance Releases Only</mark>: Run QA scripts from branch `stork_vA_B` of `qa-dhcp` instead of `master`.

## Pre-Release Preparation

Some of these checks and updates can be made before the actual freeze.

1. [ ] <mark>Security Release Only</mark>: Should be done before the first security fix is merged. Make sure mirroring is turned off for both Github and Gitlab [here](https://gitlab.isc.org/isc-projects/kea/-/settings/repository#js-push-remote-settings). To turn it off, run QA script [toggle-repo-mirroring.py](https://gitlab.isc.org/isc-private/q>
   Example command: `GITLAB_TOKEN='...' ./toggle-repo-mirroring.py --off isc-projects/kea`.
1. [ ] <mark>Security Release Only</mark>: The release will be done from the isc-private/kea. Rebase the private branch you are releasing on the public branch. If there are changes after the rebase, push them. If there are conflicts, you may ask developers for help on resolving them.
1. [ ] Check Jenkins and Gitlab CI results:
    1. [ ] Check Jenkins jobs report: [report](https://jenkins.aws.isc.org/job/stork/job/tests-report/Stork_20Tests_20Report/).
    1. [ ] Check [the latest pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/latest). <mark>Stable and Maintenance Releases</mark>: check [the stable pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/stork_v*_*/latest) instead (draft link, edit).
        - Sometimes, some jobs fail because of infrastructure problems. You can click Retry on the pipeline page, or retry jobs individually to see if the errors go away.
    1. [ ] Upload necessary changes and fixes.
1. [ ] Do some quick checks on https://demo.stork.isc.org/. There should be an old version deployed at this time, but there may be bugs worth pointing out to the Stork team, or other bugs that affect the normal release workflow to fix.
    - Login with admin:admin using LDAP credentials.
    - Stork should have not requested for the credentials to be changed after logging in.
    - Do any error notifications appear when you load the page?
    - Hover over the logo in the top left corner, and check that the tooltip shows the expected version and build date.
    - Check that the page is loading outside VPN.
1. Check if ReadTheDocs can build Stork documentation.
    1. [ ] Check if [the latest build](https://app.readthedocs.org/projects/stork/builds/?version__slug=latest) was successful and if its time matches the merge time of the release changes. <mark>Stable and Maintenance Releases</mark>: check [the stable build](https://app.readthedocs.org/projects/stork/builds/?version__slug=v2.4.6) instead (draft link, edit).
    1. If not, trigger rebuilding docs on [readthedocs.org](https://app.readthedocs.org/projects/stork/builds) and wait for the build to complete.
1. Prepare release notes.
    1. [ ] Create a draft of the release notes on the [Stork GitLab wiki](https://gitlab.isc.org/isc-projects/stork/-/wikis/home). It should be created under [the Releases directory](https://gitlab.isc.org/isc-projects/stork/-/wikis/Releases), like this one: https://gitlab.isc.org/isc-projects/stork/-/wikis/Releases/Release-notes-2.0.0.
    1. [ ] Notify @tomek that the draft is ready to be redacted. Wait for that to be done.
    1. [ ] Notify support that release notes are ready for review. To avoid conflicts in edits wait with next step after review is done. Due to the time difference, please do this at least 36 hours before the planned release.

The following steps may involve changing files in the repository.

1. [ ] Prepare release changes. Run QA script [stork/release/update-code-for-release.py](https://gitlab.isc.org/isc-private/qa-dhcp/-/blob/master/stork/release/update-code-for-release.py).
    * e.g. `GITLAB_TOKEN='...' ./update-code-for-release.py --release-date 'Feb 07, 2030' --version=1.2.0 --repo-dir=/home/wlodek/stork`
    * [ ] <mark>Stable or Maintenance Releases Only</mark>: please run from `stork_v*_*` branch of `qa-dhcp`.
1. [ ] If any systems were added or removed in Jenkins since the latest release, update `doc/user/compatible-systems.csv`. The columns that could be modified are: `Unit Tests`, `System Tests`, `Installation & Upgrade & Run (with systemd)`. The legend for what `X`, `D`, `U` mean is in `doc/user/install.rst` or below the table at https://stork.readthedocs.io/en/latest/install.html#compatible-systems.
1. [ ] Check correctness of changes applied and commit changes by rerunning `./update-code-for-release.py` with `--upload-only` option (script will skip all steps related to applying code changes).
    * e.g.  `GITLAB_TOKEN='...' ./update-code-for-release.py --repo-dir=/home/wlodek/stork --upload-only`.
    This will: commit and push changes, create issue and MR for release changes.
1. [ ] Conduct review process on release changes and merge the MR.
1. [ ] Wait for Jenkins jobs and pipelines to conclude, check their status:
    1. [ ] Check Jenkins jobs report: [report](https://jenkins.aws.isc.org/job/stork/job/tests-report/Stork_20Tests_20Report/).
    1. [ ] Check [the latest pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/latest).
        - Sometimes, some jobs fail because of infrastructure problems. You can click Retry on the pipeline page, or retry jobs individually to see if the errors go away.
        - If packaging jobs failed, it is likely that the `pkg` Jenkins job also failed. Re-run it in that case. It is responsible for uploading packages to Nexus. Released packages in Nexus are required for testing.
1. [ ] Test that uploading to Cloudsmith works.
    1. Go to [the latest pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/latest).
    1. Run `upload_test_packages`.
    1. Run `upload_test_packages_hooks`.
    1. Wait for the jobs to complete.
    1. Check that the packages were uploaded to Cloudsmith: https://cloudsmith.io/~isc/repos/stork-testing/packages/. There should be `18 == 2 (amd + arm) * 3 (apk + deb + rpm) * 2 (agent + server + ldap)` total packages.
    1. If there were any errors, investigate and fix.
1. [ ] Request sanity checks from the team. Run QA script [stork/release/request-sanity-checks.py](https://gitlab.isc.org/isc-private/qa-dhcp/-/blob/master/stork/release/request-sanity-checks.py).
    * Example command: `GITLAB_TOKEN='...' ./request-sanity-checks.py`
    * Fallback if it does not work:
        1. Go to [the latest pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/latest).
        1. Get the job IDs from `tarball`, `tarball_hooks`, `packages`, `packages_hooks` jobs.
        1. Create an issue with the following text after filling `{var}`s. Artifact `{var}`s are of the form `https://gitlab.isc.org/isc-projects/stork/-/jobs/{id}/artifacts/browse`.
```
We are now at step SANITY CHECKS of Stork {stork_version} rc{rc}.

You can do sanity checks according to the steps below:

1. Get the tarball and check it - build with `rake build`, run tests with `rake unittest:backend`, `rake unittest:ui`, `rake systemtest`, etc.
2. Get the apk, deb & rpm packages, install them.
3. Start Stork locally from tarball, packages, or demo, and follow the steps from the demo wiki: https://gitlab.isc.org/isc-projects/stork/-/wikis/Demo.

Before starting, please state what you are checking in a thread/discussion (not as comment) in the sanity checks issue in GitLab: {gl_issue}

When you finish a check, state in the same thread/discussion what the result is.
This way we know what is covered upfront and we can avoid repeating ourselves.

* tarball: {tarball_artifacts}
* apk, deb & rpm packages: {packages_artifacts}

Hooks:
  * tarball: {hooks_tarballs_artifacts}
  * apk, deb & rpm packages: {hooks_packages_artifacts}

Release notes: {release_notes}
```
1. [ ] If reported issues require immediate fixes and respin, please follow standard procedure of creating issue and review.
    1. [ ] Close current sanity check issue.
1. [ ] If reported issues do NOT require respin, proceed to the next section: [Releasing Tarballs and Packages](#releasing-tarballs-and-packages).

## Releasing Tarballs and Packages

1. [ ] Finish release notes, paste there the change log.
1. Deploy source tarball & release notes to repo.isc.org.
   1. Go to [the latest pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/latest).
   1. Run `upload_to_repo`.
   1. Run `upload_to_repo_hooks`.
   1. Wait for the jobs to complete.
   1. [ ] Check that the tarballs were uploaded to repo.isc.org:/data/shared/sweng/stork/releases/.
1. Upload packages to cloudsmith.
    1. Go to [the latest pipeline](https://gitlab.isc.org/isc-projects/stork/-/pipelines/latest).
    1. Run `upload_packages`.
    1. Run `upload_packages_hooks`.
    1. Wait for the jobs to complete.
    1. [ ] Check that the packages were uploaded to Cloudsmith: https://cloudsmith.io/~isc/repos/stork-dev/packages/. There should be `18 == 2 (amd + arm) * 3 (apk + deb + rpm) * 2 (agent + server + ldap)` total packages. <mark>Stable or Maintenance Releases Only</mark>: check https://cloudsmith.io/~isc/repos/stork/packages instead.
1. [ ] Sign the tarballs. Run QA script [stork/release/sign-tarballs.sh](https://gitlab.isc.org/isc-private/qa-dhcp/-/blob/master/stork/release/sign-tarballs.sh).
    * Example command: `./sign-tarballs.sh 1.2.0 wlodek 0259A33B5F5A3A4466CF345C7A5E084CACA51884`
    * To get the fingerprint, run `gpg --list-keys wlodek@isc.org`.
    * Fallback if it does not work:
        1. Download the tarballs from `repo.isc.org:/data/shared/sweng/stork/releases/x.y.z/stork*-x.y.z.tar.gz`.
        1. Sign them.
        1. Upload the public signature at `/data/shared/sweng/stork/releases/x.y.z/stork*-x.y.z.tar.gz.asc`.
1. [ ] Log in to repo.isc.org and publish the final tarball to the public FTP site using the make-available script.
    * Example command: `make-available --public /data/shared/sweng/stork/releases/1.2.0`
    * For more information use `--debug` option.
    * To overwrite existing content, use `--force` option.
    * If you did a mistake, contact ASAP someone from the ops team to remove incorrectly uploaded tarballs.
1. [ ] Clean up test packages on Cloudsmith since they are no longer required.
    1. Go to https://cloudsmith.io/~isc/repos/stork-testing/packages/.
    1. Click the checkbox that checks all packages.
    1. Click the red trash-can icon that says `Delete (remove) completely.`.
1. Create a signed tag. Run QA script [sign-tag.py](https://gitlab.isc.org/isc-private/qa-dhcp/-/blob/master/kea/build/sign-tag.py).
    1. [ ] Once for `isc-projects/stork`.
    1. [ ] Once for `isc-projects/stork-hook-ldap`.
    * Example command: `./sign-tag.py --project isc-projects/stork --tag v1.2.0 --branch master --key 0259A33B5F5A3A4466CF345C7A5E084CACA51884`
    * To get the fingerprint, run `gpg --list-keys wlodek@isc.org`.
1. [ ] Create the Gitlab release. Run QA script [stork/release/add-tag-and-release.py](https://gitlab.isc.org/isc-private/qa-dhcp/-/blob/master/stork/release/add-tag-and-release.py). (Connection to repo.isc.org will be required.)
    * Example command: `GITLAB_TOKEN='...' ./add-tag-and-release.py`
    * Fallback if it does not work:
        1. Tag the selected commit at the top with the proper version tag in the `vx.y.z` format on. Put there a link to the release notes page (e.g. https://gitlab.isc.org/isc-projects/stork/-/wikis/releases/Release-notes-1.2.0) and a link to the ARM (e.g. https://stork.readthedocs.io/en/v1.2.0/). Then click on `Create release`. Do this for:
            1. https://gitlab.isc.org/isc-projects/stork/-/tags.
            1. https://gitlab.isc.org/isc-projects/stork-hook-ldap/-/tags.

        1. Send a message to [the Stork channel](https://mattermost.isc.org/isc/channels/stork). Include the path to the release artifacts and a checklist. Here is a template:
            ```
            #### Stork 1.2.0 is ready to be published.

            The tarballs, the signature, and the release notes are at:

            - `repo.isc.org:/data/shared/sweng/stork/releases/1.2.0`
            - `ftp://ftp.isc.org/isc/stork/1.2.0`
            - https://downloads.isc.org/isc/stork/1.2.0

            ##### Checksums:

            SHA512 (/data/shared/sweng/stork/releases/1.2.0/stork-1.2.0.tar.gz) = ...
            SHA512 (/data/shared/sweng/stork/releases/1.2.0/stork-server-ldap-1.2.0.tar.gz) = ...

            ##### Checklist:

            - [x] Sign tarballs. (release engineer)
            - [x] Upload tarballs, signatures, and release notes to repo.isc.org. (release engineer)
            - [x] Upload packages to https://cloudsmith.io/~isc/repos/stork/packages/. (release engineer)
            - [ ] Publish links to downloads on ISC website. (marketing)
            - [ ] Write release email to *stork-users*. (marketing)

            Code freeze is over.
            ```
1. [ ] <mark>Latest Stable Release Only</mark>: Recreate the `stable` tag. Go to [the stable tag](https://gitlab.isc.org/isc-projects/stork/-/tags/stable), click `Delete tag`, then `New tag`, `Tag name`: `stable`, `Create from`: `stork_v*_*`.
1. [ ] <mark>Security Release Only</mark>: Wait for public disclosure.
1. [ ] <mark>Security Release Only</mark>: After public disclosure, sync release branches from Stork private repositories into Stork public.
1. [ ] Update docs on https://app.readthedocs.org/projects/stork/.
    1. Click the triple dot button on the `latest` build -> click `Rebuild version`. This is really a workaround for RTD to pull the repo and discover the new tag.
    1. Go to `Versions` -> `Add version` -> find the tag name in the dropdown menu -> check `Active` -> click `Update version`. Wait for the build to complete.
    1. [ ] <mark>Latest Stable Release Only</mark>: change default version:
        1. Go to `Settings` -> `Default version:` -> choose the new version as default.
        1. Check that https://stork.readthedocs.io/ redirects to the new version.
    1. [ ] <mark>Latest Stable Release Only</mark>: Rebuild the `stable` version. Go to [the stable build](https://app.readthedocs.org/projects/kea/builds/?version__slug=stable), click `Rebuild version`.

1. [ ] <mark>Stable or Maintenance Releases Only</mark>: follow [those instructions](https://gitlab.isc.org/isc-private/stork/-/wikis/Release-Procedure#update-the-public-stork-demo) to update public demo

1. [ ] <mark>Stable Release Only</mark>: Update the [the Stork Quickstart Guide](https://kb.isc.org/docs/stork-quickstart-guide).

1. [ ] Contact the Marketing team, and assign this ticket to a member who will continue working on this release.

## Marketing

1. [ ] Publish links to downloads on the ISC website.
1. [ ] <mark>Stable or Maintenance Releases Only</mark>: Write release email to [kea-announce](https://lists.isc.org/pipermail/kea-announce/).
1. [ ] <mark>Stable or Maintenance Releases Only</mark>: Announce release to support subscribers using the read-only Kea Announce queue.
1. [ ] Write email to [stork-users](https://lists.isc.org/pipermail/stork-users/).
1. [ ] Announce on social media.
1. [ ] <mark>Stable or Maintenance Releases Only</mark>: Write blog article.
1. [ ] Check if [the Stork website page](https://www.isc.org/stork/) needs updating.
1. [ ] Contact the Support team, and assign this ticket to a member who will continue working on this release.

## Support

1. [ ] Update tickets in case of waiting for support customers.

## QA

1. [ ] <mark>Security Release Only</mark>: Mirroring can be turned back on for both Github and Gitlab. You an check it [here](https://gitlab.isc.org/isc-projects/stork/-/settings/repository#js-push-remote-settings). To turn it on, run QA script [toggle-repo-mirroring.py](https://gitlab.isc.org/isc-private/qa-dhcp/-/blob/master/scripts/toggle-repo-mirroring.py) \
   Example command: `GITLAB_TOKEN='...' ./toggle-repo-mirroring.py --on isc-projects/stork`.
1. [ ] Close this ticket.
