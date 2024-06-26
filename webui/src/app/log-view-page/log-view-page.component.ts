import { Component, OnInit } from '@angular/core'
import { ActivatedRoute } from '@angular/router'
import { ServicesService } from '../backend/api/api'
import { getErrorMessage } from '../utils'

/**
 * Component providing a simple log viewer for remote log files.
 *
 * The component sends a query to tail the log having the specified
 * ID. The tail of the returned log is shown in the text box. The
 * severities of the log messages are highlighted for each message.
 *
 * Currently, the log viewer is not following the changes in the file.
 * However, a refresh button is provided which sends a request to
 * get the updated log tail.
 */
@Component({
    selector: 'app-log-view-page',
    templateUrl: './log-view-page.component.html',
    styleUrls: ['./log-view-page.component.sass'],
})
export class LogViewPageComponent implements OnInit {
    maxLengthChunk = 4000
    maxLength = this.maxLengthChunk

    appId: number
    appName: string
    appType: string
    appTypeCapitalized: string
    private _logId: number
    contents: string[]
    data: any

    /**
     * Indicates if the new request for data has been sent and the
     * response is under way. When set to false, the spinner is
     * activated to indicate that the data is loading.
     */
    loaded = false
    loadingError = null

    /**
     * Constructor
     *
     * @param route object used to get the requested log id
     * @param servicesApi object used in communication with the server
     */
    constructor(
        private route: ActivatedRoute,
        private servicesApi: ServicesService
    ) {}

    /**
     * Sends initial request for log tail
     */
    ngOnInit(): void {
        this.route.paramMap.subscribe((params) => {
            const logIdStr = params.get('id')
            if (logIdStr) {
                this._logId = parseInt(logIdStr, 10)
                this.fetchLogTail()
            }
        })
    }

    /**
     * Sends the request to the server to fetch the tail of the log file
     *
     * The log identifier must exist before the request is sent. If the
     * response is ok, the panel title is set and it includes the path
     * to the log and the machine IP address and port. The log text box
     * is filled with the data and the loaded flag is set to true to
     * disable the spinner.
     */
    private fetchLogTail() {
        this.loaded = false
        this.servicesApi.getLogTail(this._logId, this.maxLength).subscribe(
            (data) => {
                // store received data
                this.data = data

                // Set other data.
                this.appId = data.appId
                this.appName = data.appName
                this.appType = data.appType
                if (this.appType.length > 1) {
                    this.appTypeCapitalized = this.appType.charAt(0).toUpperCase() + this.appType.slice(1)
                }
                // Fill the text box with the log contents.
                this.contents = data.contents

                // Disable the spinner.
                this.loaded = true

                // handle error case
                if (data.error) {
                    this.loadingError = data.error
                } else {
                    this.loadingError = null
                }
            },
            (err) => {
                this.loaded = true
                const msg = getErrorMessage(err)
                this.loadingError = msg
            }
        )
    }

    /**
     * Refreshes the log.
     *
     * This action is triggered when the refresh button is clicked.
     */
    refreshLog() {
        if (!this.loaded) {
            return
        }
        this.fetchLogTail()
    }

    /**
     * Increases the size of the log to be fetched and re-fetches the log.
     *
     * This action is triggered when the plus button is clicked.
     */
    fetchMoreLog() {
        if (!this.loaded) {
            return
        }
        this.maxLength += this.maxLengthChunk
        this.fetchLogTail()
    }

    /**
     * Decreases the size of the log to be fetched and re-fetches the log.
     *
     * This action is triggered when the plus button is clicked. The action is
     * no-op if the max length is already equal to or less than 4000 bytes.
     */
    fetchLessLog() {
        if (!this.loaded) {
            return
        }
        if (this.maxLength > this.maxLengthChunk) {
            this.maxLength -= this.maxLengthChunk
            this.fetchLogTail()
        }
    }

    /**
     * Parses a single line of the log
     *
     * This function attempts to locate the severity of the given log message and
     * splits the single line into 3, one for the part before the severity, second
     * for severity and the third part containing the text after severity.
     *
     * @param line log message line.
     * @returns array of strings containing single partitioned log message. If the
     *          severity hasn't been found, the returned array comprises a single
     *          element holding the entire log message.
     */
    parseLogLine(line): string[] {
        return line.split(/(FATAL|ERROR|WARN|INFO|DEBUG)/)
    }

    /**
     * Returns the color to be used for the given severity.
     *
     * @param block part of the log message comprising severity.
     *
     * @returns name of the color in which the message with the given severity should
     *          be highlighted, i.e. red for fatal and error messages, orange for
     *          warnings, cyan for info and white for other error messages.
     */
    logSeverityColor(block) {
        switch (block) {
            case 'ERROR':
            case 'FATAL':
                return 'var(--red-500)'
            case 'WARN':
                return 'var(--orange-400)'
            case 'INFO':
                return 'cyan'
            default:
                return 'white'
        }
        return 'white'
    }
}
