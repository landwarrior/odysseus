@extends('layouts.app')

@section('content')
<div class="container">
  <div class="row justify-content-center">
    <div class="col-md-12">
      @if (session('not_admin'))
      <div class="alert alert-danger alert-dismissible fade show" role="alert">
        {{ __('messages.not_admin') }}
        <button type="button" class="close" data-dismiss="alert" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      @endif
      <div class="card">
        <div class="card-header">Dashboard</div>
        <div class="card-body">
          @if (session('status'))
          <div class="alert alert-success" role="alert">
            {{ session('status') }}
          </div>
          @endif
          <div class="container">
            <div class="row justify-content-center links text-center">
              <div class="col-md-3">
                <a href="/result">{{ __('messages.result.index') }}</a>
              </div>
            </div>
          </div>
        </div>
        @if ($is_admin)
        @include('admin')
        @endif
      </div>
    </div>
  </div>
</div>
@endsection
