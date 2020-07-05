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
  @if(session('no_regist'))
  <div class="flash-msg alert alert-info fade show" role="alert">
    {{ __('messages.no_regist') }}
  </div>
  @endif
  <div class="row">
    <div class="col-12">
      <div class="card">
        <div class="card-header">{{ __('messages.project.list') }}</div>
        <div class="card-body">
          <table class="table table-hover">
            <thead class="thead-light">
              <tr class="text-center">
                <th scope="col" class="break">{{ __('messages.project.no') }}</th>
                <th scope="col" class="break">{{ __('messages.project.name') }}</th>
                <th scope="col" class="break">{{ __('messages.project.order_amount') }}</th>
                <th scope="col" class="break">{{ __('messages.project.from') }}</th>
                <th scope="col" class="break">{{ __('messages.project.to') }}</th>
                <th scope="col" class="break">{{ __('messages.project.assign') }}</th>
              </tr>
            </thead>
            <tbody>
              @if(count($projects) == 0)
              <tr class="text-center">
                <td colspan="5">{{ __('messages.project.not_any') }}</td>
              </tr>
              @endif
              @foreach($projects as $project)
              <tr class="text-center" data-href="/project/{{ $project->project_no }}">
                <td class="break">{{ $project->project_no }}</td>
                <td class="break">{{ $project->name }}</td>
                <td class="text-right break">￥ {{ number_format($project->order_amount) }}</td>
                <td class="break">@if($project->from_date){{ $project->from_date->format('Y/m/d') }}@endif</td>
                <td class="break">@if($project->to_date){{ $project->to_date->format('Y/m/d') }}@endif</td>
                <td class="text-center break">
                  <button type="button" class="btn btn-primary" data-href="/projecthr/{{ $project->project_no }}">
                    <svg width="1em" height="1em" viewBox="0 0 16 16" class="bi bi-person-plus" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
                      <path fill-rule="evenodd" d="M11 14s1 0 1-1-1-4-6-4-6 3-6 4 1 1 1 1h10zm-9.995-.944v-.002.002zM1.022 13h9.956a.274.274 0 0 0 .014-.002l.008-.002c-.001-.246-.154-.986-.832-1.664C9.516 10.68 8.289 10 6 10c-2.29 0-3.516.68-4.168 1.332-.678.678-.83 1.418-.832 1.664a1.05 1.05 0 0 0 .022.004zm9.974.056v-.002.002zM6 7a2 2 0 1 0 0-4 2 2 0 0 0 0 4zm3-2a3 3 0 1 1-6 0 3 3 0 0 1 6 0zm4.5 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1H13V5.5a.5.5 0 0 1 .5-.5z"/>
                      <path fill-rule="evenodd" d="M13 7.5a.5.5 0 0 1 .5-.5h2a.5.5 0 0 1 0 1H14v1.5a.5.5 0 0 1-1 0v-2z"/>
                    </svg>
                  </button>
                </td>
              </tr>
              @endforeach
            </tbody>
          </table>
          <div class="row">
            <div class="col-6">
              <a href="/project/create" class="btn btn-primary" role="button">{{ __('messages.create') }}</a>
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
