<div class="container">
  <div class="row">
    <div class="col-md-12">
      @include('message')
      <div class="card">
        <div class="card-header">{{ __('messages.result.index') }}</div>
        <div class="card-body">
          <div class="row" style="padding-bottom: 20px">
            <div class="col-6">
              <a href="/result/{{ $project_no }}?date={{ $last_month }}" class="btn btn-lg btn-outline-secondary">
                <svg width="1em" height="1em" viewBox="0 0 16 16" class="bi bi-chevron-left" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
                  <path fill-rule="evenodd" d="M11.354 1.646a.5.5 0 0 1 0 .708L5.707 8l5.647 5.646a.5.5 0 0 1-.708.708l-6-6a.5.5 0 0 1 0-.708l6-6a.5.5 0 0 1 .708 0z"/>
                </svg>
                {{ __('messages.result.last') }}
              </a>
            </div>
            <div class="col-6 text-right">
              <a href="/result/{{ $project_no }}?date={{ $next_month }}" class="btn btn-lg btn-outline-secondary">
                {{ __('messages.result.next') }}
                <svg width="1em" height="1em" viewBox="0 0 16 16" class="bi bi-chevron-right" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
                  <path fill-rule="evenodd" d="M4.646 1.646a.5.5 0 0 1 .708 0l6 6a.5.5 0 0 1 0 .708l-6 6a.5.5 0 0 1-.708-.708L10.293 8 4.646 2.354a.5.5 0 0 1 0-.708z"/>
                </svg>
              </a>
            </div>
          </div>
          <form action="/result/{{ $project_no }}" method="post">
            <input type="hidden" name="_method" value="PUT">
            @csrf
            <table class="table">
              <thead>
                <tr class="text-center">
                  <th>{{ __('messages.result.target_date') }}</th>
                  @foreach($targets as $target)
                  <th>{{ $process_map[$target->process_id] }}</th>
                  @endforeach
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td><input type="text" name="target_date" class="datepicker form-control text-center" value=""></td>
                  @foreach($targets as $target)
                  <td>
                    <input type="text" name="result[{{ $target->process_id }}]" class="form-control text-right" value="" placeholder="{{ __('messages.result.hour') }}">
                  </td>
                  @endforeach
                </tr>
              </tbody>
              <tfoot class="table-sm">
                @foreach($results as $target_date => $result)
                <tr>
                  <td class="text-center">{{ $target_date }}</td>
                  @foreach($targets as $target)
                  <td class="text-right" style="padding-right: 15px">
                    @if(isset($result[$target->process_id]))
                    {{ $result[$target->process_id] }} {{ __('messages.result.hour') }}
                    @endif
                  </td>
                  @endforeach
                </tr>
                @endforeach
              </tfoot>
            </table>
            <div class="row">
              <div class="col-6">
                <button type="submit" class="btn btn-lg btn-outline-primary">{{ __('messages.register') }}</button>
              </div>
              <div class="col-6 text-right">
                <a href="/result" class="btn btn-lg btn-outline-secondary">{{ __('messages.back') }}</a>
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</div>
@include('result.datepicker')
