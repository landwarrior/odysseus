<div class="container">
  <div class="row">
    <div class="col-md-12">
      @include('project.message')
      <div class="card">
        <div class="card-header">{{ __('messages.project.create') }}</div>
        <div class="card-body">
          @if($target == 'store')
          <form action="/project" method="post">
          @elseif($target == 'update')
          <form action="/project/{{ $project->project_no }}" method="post">
            <input type="hidden" name="_method" value="PUT">
          @endif
            @csrf
            <div class="row">
              <div class="col-lg-3">
                <div class="form-group">
                  @if ($target == 'store')
                  <div class="form-label-group">
                    <input type="text" id="project_no" class="form-control @error('project_no') is-invalid @enderror" name="project_no" value="{{ old('project_no') }}" required placeholder="{{ __('messages.project.no') }}">
                    <label for="project_no">{{ __('messages.project.no') }}</label>
                  </div>
                  @else
                  <div class="form-label-group">
                    <input type="text" id="project_no" class="form-control" name="project_no" value="{{ $project->project_no }}" readonly placeholder="{{ __('messages.project.no') }}">
                    <label for="project_no">{{ __('messages.project.no') }}</label>
                  </div>
                  @endif
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="order_amount" class="form-control text-right numFmt @error('order_amount') is-invalid @enderror" name="order_amount" value="{{ old('order_amount', $project->order_amount) }}" placeholder="{{ __('messages.project.order_amount') }}">
                    <label for="order_amount">{{ __('messages.project.order_amount') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="from_date" class="form-control text-center datepicker @error('from_date') is-invalid @enderror" name="from_date"
                           value="{{ old('from_date', $project->from_date ? $project->from_date->format('Y/m/d') : '') }}" placeholder="{{ __('messages.project.from') }}">
                    <label for="from_date">{{ __('messages.project.from') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-3">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="to_date" class="form-control text-center datepicker @error('to_date') is-invalid @enderror" name="to_date"
                           value="{{ old('to_date', $project->to_date ? $project->to_date->format('Y/m/d') : '') }}" placeholder="{{ __('messages.project.to') }}">
                    <label for="to_date">{{ __('messages.project.to') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="project_name" class="form-control @error('project_name') is-invalid @enderror" name="project_name" value="{{ old('project_name', $project->name) }}" required placeholder="{{ __('messages.project.name') }}">
                    <label for="project_name">{{ __('messages.project.name') }}</label>
                  </div>
                </div>
              </div>
            </div>
            <div class="row">
              <div class="col-6">
                <button type="submit" class="btn btn-lg btn-outline-primary">{{ __('messages.register') }}</button>
              </div>
              <div class="col-6 text-right">
                <a href="/project" class="btn btn-lg btn-outline-secondary">{{ __('messages.back') }}</a>
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</div>
@include('layouts.datepicker')
