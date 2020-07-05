@extends('layouts.app')

@section('content')
<div class="container">
  @if(session('registered'))
  <div class="flash-msg alert alert-success fade show" role="alert">
    {{ __('messages.registered') }}
  </div>
  @endif
  @if(session('deleted'))
  <div class="flash-msg alert alert-danger fade show" role="alert">
    {{ __('messages.deleted') }}
  </div>
  @endif
  <div class="row">
    <div class="col-12">
      <div class="card">
        <div class="card-header">{{ __('messages.hr.list') }}</div>
        <div class="card-body">
          <table class="table table-hover">
            <thead class="thead-light">
              <tr class="text-center">
                <th scope="col" style="width: 150px">{{ __('messages.hr.hr_cd') }}</th>
                <th scope="col">{{ __('messages.hr.user_name') }}</th>
                <th scope="col" style="width: 180px">{{ __('messages.hr.name_kana') }}</th>
                <th scope="col" style="width: 140px">{{ __('messages.hr.is_admin') }}</th>
                <th scope="col" style="width: 250px">{{ __('messages.hr.bp_name') }}</th>
              </tr>
            </thead>
            <tbody>
              @foreach($humans as $hr)
              <tr class="text-center" data-href="/hr/{{ $hr->hr_cd }}">
                <td>{{ $hr->hr_cd }}</td>
                <td>{{ $hr->user_name }}</td>
                <td>{{ $hr->name_kana }}</td>
                <td>@if($hr->is_admin) ✔ @endif</td>
                <td></td>
              </tr>
              @endforeach
            </tbody>
          </table>
          <div class="row">
            <div class="col-6">
              <a href="/hr/create" class="btn btn-primary" role="button">{{ __('messages.create') }}</a>
            </div>
            <div class="col-6 text-right">
              <a href="/" class="btn btn-outline-secondary" role="button">{{ __('messages.back') }}</a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
@endsection
