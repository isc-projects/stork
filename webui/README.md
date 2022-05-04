## @

### Building

To install the required dependencies and to build the typescript sources, run:

```
npm install
npm run build
```

### Publishing

First build the package, then run `npm publish`.

### Consuming

Navigate to the project folder and run one of the following commands:

_publishing:_

```
npm install @ --save
```

_without publishing (not recommended):_

```
npm install PATH_TO_GENERATED_PACKAGE --save
```

_using `npm link`:_

In PATH_TO_GENERATED_PACKAGE:

```
npm link
```

In the project:

```
npm link
```

**Note for Windows users:** The Angular CLI has trouble using linked npm packages.
Please refer to this issue https://github.com/angular/angular-cli/issues/8284 for a solution/workaround.
Published packages are not affected by this issue.

#### General Usage

In an Angular project:

```
// without configuring providers
import { ApiModule } from '';
import { HttpClientModule } from '@angular/common/http';


@NgModule({
    imports: [
        ApiModule,
        // make sure to import the HttpClientModule in the AppModule only,
        // see https://github.com/angular/angular/issues/20575
        HttpClientModule
    ],
    declarations: [ AppComponent ],
    providers: [],
    bootstrap: [ AppComponent ]
})
export class AppModule {}
```

```
// configuring providers
import { ApiModule, Configuration, ConfigurationParameters } from '';

export function apiConfigFactory (): Configuration => {
  const params: ConfigurationParameters = {
    // set configuration parameters here.
  }
  return new Configuration(params);
}

@NgModule({
    imports: [ ApiModule.forRoot(apiConfigFactory) ],
    declarations: [ AppComponent ],
    providers: [],
    bootstrap: [ AppComponent ]
})
export class AppModule {}
```

```
import { DefaultApi } from '';

export class AppComponent {
	 constructor(private apiGateway: DefaultApi) { }
}
```

Note: The ApiModule is restricted to being instantiated one time, app-wide.
This is to ensure that all services are treated as singletons.

#### Using Multiple Swagger files/APIs/ApiModules

To use multiple `ApiModules` generated from different swagger files,
create an alias name when importing the modules
to avoid naming conflicts:

```
import { ApiModule } from 'my-api-path';
import { ApiModule as OtherApiModule } from 'my-other-api-path';
import { HttpClientModule } from '@angular/common/http';


@NgModule({
  imports: [
    ApiModule,
    OtherApiModule,
    // make sure to import the HttpClientModule in the AppModule only,
    // see https://github.com/angular/angular/issues/20575
    HttpClientModule
  ]
})
export class AppModule {

}
```

### Setting the Service Base Path

If different than the generated base path, the base path to the service can be provided during app bootstrap.

```
import { BASE_PATH } from '';

bootstrap(AppComponent, [
    { provide: BASE_PATH, useValue: 'https://your-web-service.com' },
]);
```

or

```
import { BASE_PATH } from '';

@NgModule({
    imports: [],
    declarations: [ AppComponent ],
    providers: [ provide: BASE_PATH, useValue: 'https://your-web-service.com' ],
    bootstrap: [ AppComponent ]
})
export class AppModule {}
```

#### Using @angular/cli

First, extend the `src/environments/*.ts` files by adding the corresponding base path:

```
export const environment = {
  production: false,
  API_BASE_PATH: 'http://127.0.0.1:8080'
};
```

In src/app/app.module.ts:

```
import { BASE_PATH } from '';
import { environment } from '../environments/environment';

@NgModule({
  declarations: [
    AppComponent
  ],
  imports: [ ],
  providers: [{ provide: BASE_PATH, useValue: environment.API_BASE_PATH }],
  bootstrap: [ AppComponent ]
})
export class AppModule { }
```
