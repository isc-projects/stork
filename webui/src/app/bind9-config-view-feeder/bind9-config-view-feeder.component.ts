import { Component, DestroyRef, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { Bind9FormattedConfig, ServicesService } from '../backend'
import { MessageService } from 'primeng/api'
import { TextFileViewerComponent } from '../text-file-viewer/text-file-viewer.component'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { CommonModule } from '@angular/common'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { catchError, EMPTY, finalize, Subject, switchMap, takeUntil, tap } from 'rxjs'

/**
 * A component that fetches the BIND 9 configuration from the server.
 */
@Component({
    selector: 'app-bind9-config-view-feeder',
    imports: [CommonModule, ProgressSpinnerModule, TextFileViewerComponent],
    templateUrl: './bind9-config-view-feeder.component.html',
    styleUrl: './bind9-config-view-feeder.component.sass',
})
export class Bind9ConfigViewFeederComponent implements OnInit {
    /**
     * The ID of the daemon whose configuration is being fetched.
     */
    @Input({ required: true }) daemonId: number

    /**
     * The type of the file to be displayed. The server uses this
     * selection to determine which file contents to return.
     */
    @Input({ required: true }) fileType: 'config' | 'rndc-key'

    /**
     * Sets the flag indicating that the component should fetch the
     * BIND 9 configuration.
     */
    @Input({ required: true }) set active(active: boolean) {
        if (active && !this._loaded) {
            this.updateConfig(false)
        }
    }

    /**
     * An event emitter that emits the new value of the configuration
     * when the configuration has been updated.
     */
    @Output() configChange = new EventEmitter<Bind9FormattedConfig>()

    /**
     * The configuration to be displayed.
     */
    config: Bind9FormattedConfig | null = null

    /**
     * Indicates if the configuration has been loaded.
     *
     * It is used to display a loading spinner while the configuration is being loaded.
     */
    loading: boolean = false

    /**
     * Holds the flag indicating that the configuration has been loaded.
     *
     * It is used to prevent loading the configuration multiple times.
     */
    _loaded: boolean = false

    /**
     * A subject used to explicitly trigger the HTTP call to get the configuration.
     */
    private _updateTrigger$ = new Subject<{ fullConfig: boolean }>()

    /**
     * A subject used to explicitly cancel the HTTP call to get the configuration.
     */
    private _cancelTrigger$ = new Subject<void>()

    /**
     * Constructor.
     *
     * @param _servicesApi is the API service exposing the function to get the
     *        BIND 9 configuration.
     * @param _messageService is the service used to display messages to the user.
     * @param _destroyRef is the destroy ref used to destroy the subscription when
     *        the component is destroyed.
     */
    constructor(
        private _servicesApi: ServicesService,
        private _messageService: MessageService,
        private _destroyRef: DestroyRef
    ) {}

    /**
     * Lifecycle hook called after component initialization.
     *
     * It creates a subscription that fetches BIND 9 configuration when
     * triggered by the updateConfig method. The subscription is destroyed
     * when the component is destroyed. The HTTP call to get the configuration
     * can be cancelled by the cancelUpdateConfig method.
     */
    ngOnInit(): void {
        // Subscribe to the trigger. There is no need to explicitly
        // unsubscribe because takeUntilDestroyed will handle it.
        this._updateTrigger$
            .pipe(
                // Indicate that the loading is in progress before
                // making the HTTP call.
                tap(() => (this.loading = true)),
                // switchMap reacts to receiving the call parameters via
                // the trigger. It cancels any ongoing HTTP call and starts
                // a new one.
                switchMap(({ fullConfig }) =>
                    // Make the HTTP call to get the configuration.
                    this._servicesApi
                        .getBind9FormattedConfig(
                            this.daemonId,
                            fullConfig ? null : ['config'],
                            this.fileType ? [this.fileType] : null
                        )
                        .pipe(
                            // Cancel the HTTP call if the cancelTrigger is emitted.
                            takeUntil(this._cancelTrigger$),
                            tap((config) => {
                                // Receive the configuration.
                                this.config = config
                                this._loaded = true
                                // Notify the parent component that new configuration
                                // is available.
                                this.configChange.emit(config)
                            }),
                            catchError((error) => {
                                // Handle communication errors.
                                this._messageService.add({
                                    severity: 'error',
                                    summary: 'Error getting BIND 9 configuration',
                                    detail: error.message,
                                    life: 10000,
                                })
                                return EMPTY
                            }),
                            finalize(() => {
                                // This is called when received the configuration,
                                // in case of an error, or when the HTTP call is
                                // is cancelled (unsubscribed). Note that takeUntil
                                // unsubscribes from the inner subscription.
                                this.loading = false
                            })
                        )
                ),
                // Destroy the subscription when the component is destroyed.
                takeUntilDestroyed(this._destroyRef)
            )
            .subscribe()
    }

    /**
     * Fetches the configuration from the server, full or partial.
     *
     * @param fullConfig indicates if the full configuration should be fetched.
     */
    updateConfig(fullConfig: boolean): void {
        this._updateTrigger$.next({ fullConfig })
    }

    /**
     * Cancels the HTTP call to get the configuration.
     *
     * It emits a trigger to cancel the HTTP call ending the inner
     * subscription.
     */
    cancelUpdateConfig(): void {
        this._cancelTrigger$.next()
    }
}
