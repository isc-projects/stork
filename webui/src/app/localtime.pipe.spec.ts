import { LocaltimePipe } from './localtime.pipe';

describe('LocaltimePipe', () => {
  it('create an instance', () => {
    const pipe = new LocaltimePipe();
    expect(pipe).toBeTruthy();
  });
});
