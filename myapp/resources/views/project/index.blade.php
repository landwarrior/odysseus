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
        <div class="card-header">{{ __('messages.project.list') }}</div>
        <div class="card-body">
          <table class="table table-hover">
            <thead class="thead-light">
              <tr class="text-center">
                <th scope="col" style="width: 154px">{{ __('messages.project.no') }}</th>
                <th scope="col">{{ __('messages.project.name') }}</th>
                <th scope="col" style="width: 180px">{{ __('messages.project.order_amount') }}</th>
                <th scope="col" style="width: 154px">{{ __('messages.project.from') }}</th>
                <th scope="col" style="width: 154px">{{ __('messages.project.to') }}</th>
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
                <td>{{ $project->project_no }}</td>
                <td>{{ $project->name }}</td>
                <td class="text-right">￥ {{ number_format($project->order_amount) }}</td>
                <td>@if($project->from_date){{ $project->from_date->format('Y/m/d') }}@endif</td>
                <td>@if($project->to_date){{ $project->to_date->format('Y/m/d') }}@endif</td>
              </tr>
              @endforeach
            </tbody>
          </table>
          <div class="row">
            <div class="col-6">
              <a href="/project/create" class="btn btn-outline-primary" role="button">{{ __('messages.create') }}</a>
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
