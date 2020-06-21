<div class="container">
  <div class="row">
    <div class="col-md-12">
      @include('message')
      <div class="card">
        <div class="card-header">{{ __('messages.hr.create') }}</div>
        <div class="card-body">
          @if($target == 'store')
          <form action="/hr" method="post">
          @elseif($target == 'update')
          <form action="/hr/{{ $human->hr_cd }}" method="post">
            <input type="hidden" name="_method" value="PUT">
          @endif
            @csrf
            <div class="row">
              <div class="col-lg-3">
                <div class="form-group">
                  @if ($target == 'store')
                  <div class="form-label-group">
                    <input type="text" id="hr_cd" class="form-control @error('hr_cd') is-invalid @enderror" name="hr_cd" value="{{ old('hr_cd') }}" required placeholder="{{ __('messages.hr.hr_cd') }}">
                    <label for="hr_cd">{{ __('messages.hr.hr_cd') }}</label>
                  </div>
                  @else
                  <div class="form-label-group">
                    <input type="text" id="hr_cd" class="form-control" name="hr_cd" value="{{ $human->hr_cd }}" disabled placeholder="{{ __('messages.hr.hr_cd') }}">
                    <label for="hr_cd">{{ __('messages.hr.hr_cd') }}</label>
                  </div>
                  @endif
                </div>
              </div>
              <div class="col-lg-auto">
                <label for="is_admin" style="margin: 0">{{ __('messages.hr.is_admin') }}</label>
                <input type="hidden" name="is_admin" value="0">
                <input type="checkbox" class="form-control" id="is_admin" name="is_admin" value="1" style="margin: 0 auto" @if(old('is_admin', $human->is_admin)) checked @endif>
              </div>
            </div>
            <div class="row">
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="user_name" class="form-control @error('user_name') is-invalid @enderror" name="user_name" value="{{ old('user_name', $human->user_name) }}" placeholder="{{ __('messages.hr.user_name') }}">
                    <label for="user_name">{{ __('messages.hr.user_name') }}</label>
                  </div>
                </div>
              </div>
              <div class="col-lg-6">
                <div class="form-group">
                  <div class="form-label-group">
                    <input type="text" id="name_kana" class="form-control @error('from_date') is-invalid @enderror" name="name_kana" value="{{ old('name_kana', $human->name_kana) }}" placeholder="{{ __('messages.hr.name_kana') }}">
                    <label for="name_kana">{{ __('messages.hr.name_kana') }}</label>
                  </div>
                </div>
              </div>
            </div>
            <div class="row">
              <div class="col-6">
                <button type="submit" class="btn btn-lg btn-outline-primary">{{ __('messages.register') }}</button>
              </div>
              <div class="col-6 text-right">
                <a href="/hr" class="btn btn-lg btn-outline-secondary">{{ __('messages.back') }}</a>
              </div>
            </div>
          </form>
        </div>
        @if($target == 'update')
        <div class="card-header alert-danger text-center">{{ __('messages.danger_zone') }}</div>
        <div class="card-body">
          <form action="/hr/{{ $human->hr_cd }}" method="post" name="delete">
            @csrf
            <input type="hidden" name="_method" value="DELETE">
            <button type="button" id="delete-btn" class="btn btn-lg btn-block btn-danger" data-toggle="modal" data-target="#delete-modal">{{ __('messages.delete') }}</button>
          </form>
        </div>
        @endif
      </div>
    </div>
  </div>
</div>
<div class="modal fade" id="delete-modal" tabindex="-1" role="dialog" aria-labelledby="delete-modal-label" aria-hidden="true">
  <div class="modal-dialog" role="document">
    <div class="modal-content">
      <div class="modal-header alert-danger">
        <h5 class="modal-title">{{ __('messages.delete') }}</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div class="modal-body">
        {{ __('messages.hr.del_confirm') }}
        <div class="form-group">
          <div class="form-label-group">
            <input type="text" class="form-control" id="conform-delete" placeholder="{{ __('messages.hr.hr_cd') }}">
            <label for="conform-delete">{{ __('messages.hr.hr_cd') }}</label>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="btn btn-secondary" data-dismiss="modal">{{ __('messages.close') }}</button>
        <button type="button" id="delete-execute" class="btn btn-danger" disabled onclick="document.delete.submit();">{{ __('messages.delete') }}</button>
      </div>
    </div>
  </div>
</div>
@include('layouts.datepicker')
