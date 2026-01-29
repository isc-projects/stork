const { getJestConfig } = require('@storybook/test-runner')

// The default Jest configuration comes from @storybook/test-runner
const testRunnerConfig = getJestConfig()

/**
 * @type {import('@jest/types').Config.InitialOptions}
 */
module.exports = {
    ...testRunnerConfig,
    ...{
        testEnvironmentOptions: {
            // Configure Playwright for Storybook test runner.
            // See https://github.com/playwright-community/jest-playwright#configuration
            // for details.
            'jest-playwright': {
                browsers: [
                    // Use system-installed Chromium browser. It allows us to
                    // run the storybook tests with the same browser that
                    // we use for Karma unit tests. So, it is not required to
                    // install a separate browser for Storybook.
                    {
                        name: 'chromium',
                        displayName: 'Chromium',
                        launchOptions: {
                            executablePath: process.env.CHROME_BIN,
                            headless: true,
                        },
                    },
                ],
            },
        },
    },
    /** Add your own overrides below, and make sure
     *  to merge testRunnerConfig properties with your own
     * @see https://jestjs.io/docs/configuration
     */
}
