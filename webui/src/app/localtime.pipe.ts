import { Pipe, PipeTransform } from '@angular/core';
import { datetimeToLocal } from './utils';

@Pipe({
  name: 'localtime'
})
export class LocaltimePipe implements PipeTransform {

    transform(value: any, ...args: any[]): any {
        return datetimeToLocal(value);
    }

}
