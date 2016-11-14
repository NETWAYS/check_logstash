require 'spec_helper'

describe CheckLogstash do
  it ('has a version number') do 
    expect(CheckLogstash::Version).not_to be nil
  end
end
